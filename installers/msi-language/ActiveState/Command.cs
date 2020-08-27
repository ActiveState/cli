using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Text;
using System.Diagnostics;
using System.Threading;
using System.IO;
using System.Collections.Generic;

namespace ActiveState
{
    public static class Command
    {
        public static ActionResult Run(Session session, string cmd, string args, out string output)
        {
            var errBuilder = new StringBuilder();
            var outputBuilder = new StringBuilder();
            try
            {
                if (cmd == "powershell")
                {
                    cmd = Path.Combine(Environment.SystemDirectory, "WindowsPowershell", "v1.0", "powershell.exe");
                    if (!File.Exists(cmd))
                    {
                        session.Log("Did not find powershell @" + cmd);
                        cmd = "powershell.exe";
                    }
                }

                var procStartInfo = new ProcessStartInfo(cmd, args);
                // The following commands are needed to redirect the standard output.
                // This means that it will be redirected to the Process.StandardOutput StreamReader.
                procStartInfo.RedirectStandardOutput = true;
                procStartInfo.RedirectStandardError = true;
                procStartInfo.UseShellExecute = false;
                procStartInfo.StandardOutputEncoding = Encoding.UTF8;
                procStartInfo.StandardErrorEncoding = Encoding.UTF8;
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
                        errBuilder.Append("\n" + e.Data);
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
                        Process.KillProcessAndChildren(proc.Id);
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
                    outputBuilder.AppendFormat(" -- Process returned with exit code: {0}", exitCode);
                    output = outputBuilder.ToString();
                    session.Log("returning due to return code - error");
                    var title = output.Split('\n')[0];
                    if (title.Length == 0)
                    {
                        title = output;
                    }
                    if (title.Length > 50)
                    {
                        title = title.Substring(0, 50);
                    }
                    RollbarReport.Critical(
                        string.Format("failed due to return code: {0} - start: {1}", exitCode, title),
                        new Dictionary<string, object> { { "output", output }, { "err", errBuilder.ToString() }, { "cmd", cmd } }
                    );
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
                RollbarReport.Error(exceptionString);
                return ActionResult.Failure;
            }
            output = outputBuilder.ToString();
            return ActionResult.Success;
        }
    }
}
