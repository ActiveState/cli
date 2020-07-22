using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Text;
using System.Diagnostics;
using System.Threading;
using System.IO;

namespace ActiveState
{
    public enum Shell
    {
        Cmd,
        Powershell,
    }

    public static class Command
    {
        public static ActionResult Run(Session session, string cmd, Shell shell, out string output)
        {
            var outputBuilder = new StringBuilder();
            try
            {
                ProcessStartInfo procStartInfo;
                switch (shell)
                {
                    case Shell.Powershell:
                        var powershellExe = Path.Combine(Environment.SystemDirectory, "WindowsPowershell", "v1.0", "powershell.exe");
                        if (!File.Exists(powershellExe))
                        {
                            session.Log("Did not find powershell @" + powershellExe);
                            powershellExe = "powershell.exe";
                        }
                        procStartInfo = new ProcessStartInfo(powershellExe, cmd);
                        break;
                    default:
                        procStartInfo = new ProcessStartInfo("cmd.exe", "/c " + cmd);
                        break;
                }

                // The following commands are needed to redirect the standard output.
                // This means that it will be redirected to the Process.StandardOutput StreamReader.
                procStartInfo.RedirectStandardOutput = true;
                procStartInfo.RedirectStandardError = true;
                procStartInfo.UseShellExecute = false;
                // Do not create the black window.
                procStartInfo.CreateNoWindow = true;

                System.Diagnostics.Process proc = new System.Diagnostics.Process();
                proc.StartInfo = procStartInfo;

                proc.OutputDataReceived += new DataReceivedEventHandler((sender, e) =>
                {
                    var line = e.Data;
                    if (!String.IsNullOrEmpty(line))
                    {
                        session.Log("out: " + line);
                        outputBuilder.Append("\n" + line);
                    }
                });
                proc.ErrorDataReceived += new DataReceivedEventHandler((sender, e) =>
                {
                    // Prepend line numbers to each line of the output.
                    if (!String.IsNullOrEmpty(e.Data))
                    {
                        session.Log("err: " + e.Data);
                        outputBuilder.Append("\n" + e.Data);
                    }
                });
                proc.Start();

                // Asynchronously read the standard output and standard error of the spawned process.
                // This raises OutputDataReceived/ErrorDataReceived events for each line of output/errors.
                proc.BeginOutputReadLine();
                proc.BeginErrorReadLine();

                while (!proc.HasExited)
                {
                    try
                    {
                        // This is just hear to throw an InstallCanceled Exception if necessary
                        Status.ProgressBar.Increment(session, 0);
                        Thread.Sleep(200);
                    }
                    catch (InstallCanceledException)
                    {
                        session.Log("Caught install cancelled exception");
                        ActiveState.Process.KillProcessAndChildren(proc.Id);
                        output = "process got interrupted.";
                        return ActionResult.UserExit;
                    }
                }
                proc.WaitForExit();

                var exitCode = proc.ExitCode;
                session.Log(String.Format("process returned with exit code: {0}", exitCode));
                proc.Close();
                if (exitCode != 0)
                {
                    outputBuilder.Append('\x00');
                    outputBuilder.AppendFormat("Process returned with exit code: {0}", exitCode);
                    output = outputBuilder.ToString();
                    session.Log("returning due to return code - error");
                    ActiveState.RollbarHelper.Report(string.Format("returning due to return code: {0} - error", exitCode));
                    return ActionResult.Failure;
                }
            }
            catch (Exception objException)
            {
                outputBuilder.Append('\x00');
                var exceptionString = string.Format("Caught exception: {0}", objException);
                outputBuilder.Append(exceptionString);
                output = outputBuilder.ToString();
                session.Log(exceptionString);
                ActiveState.RollbarHelper.Report(exceptionString);
                return ActionResult.Failure;
            }
            output = outputBuilder.ToString();
            return ActionResult.Success;
        }
    }
}
