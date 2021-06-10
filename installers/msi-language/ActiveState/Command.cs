using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Text;
using System.Diagnostics;
using System.Threading;
using System.IO;
using System.Collections.Generic;
using System.Linq;
using System.Web.Script.Serialization;

namespace ActiveState
{
    public static class Command
    {

        public static ActionResult Run(Session session, string cmd, string args, out string output)
        {
            return RunInternal(session, cmd, args, 0, true, out output);
        }

        public static ActionResult RunWithProgress(Session session, string cmd, string args, int limit, out string output)
        {
            return RunInternal(session, cmd, args, limit, true, out output);
        }

        public static ActionResult RunAuthCommand(Session session, string cmd, string args, out string output)
        {
            return RunInternal(session, cmd, args, 0, false, out output);
        }

        private static ActionResult RunInternal(Session session, string cmd, string args, int limit, bool reportCmd, out string output)
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
                if (cmd.Contains("state.exe"))
                {
                    procStartInfo.EnvironmentVariables["VERBOSE"] = "true";
                    procStartInfo.EnvironmentVariables["ACTIVESTATE_NONINTERACTIVE"] = "true";
                    procStartInfo.EnvironmentVariables["ACTIVESTATE_CLI_DISABLE_UPDATES"] = "true";
                }
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
                        // We do not write stderr to our own log, as it comprises the progress bar output
                        session.Log("err: " + e.Data);
                        errBuilder.Append("\n" + e.Data);
                    }
                });
                proc.Start();

                // Asynchronously read the standard output and standard error of the spawned process.
                // This raises OutputDataReceived/ErrorDataReceived events for each line of output/errors.
                proc.BeginOutputReadLine();
                proc.BeginErrorReadLine();

                int count = 0;
                while (!proc.HasExited)
                {
                    try
                    {
                        // This is to update the progress bar and listen for a cancel event
                        if (count < limit) {
                            Status.ProgressBar.Increment(session, 1);
                            Thread.Sleep(150);
                        } else
                        {
                            Status.ProgressBar.Increment(session, 0);
                            Thread.Sleep(150);
                        }
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
                    session.Log("returning due to return code: {0}", exitCode);
                    if (exitCode == 11)
                    {
                        output = outputBuilder.ToString();
                        string message = FormatErrorOutput(output);
                        session.Log("Message details: {0}", message);
                        new NetworkError().SetDetails(session, message);
                    } else
                    {
                        outputBuilder.AppendFormat(" -- Process returned with exit code: {0}", exitCode);
                        output = outputBuilder.ToString();
                        var title = output.Split('\n')[0];
                        if (title.Length == 0)
                        {
                            title = output;
                        }
                        var customData = new Dictionary<string, object> { { "output", output }, { "err", errBuilder.ToString() } };
                        if (reportCmd)
                        {
                            customData["cmd"] = cmd;
                        }
                        RollbarReport.Critical(
                            string.Format("failed due to return code: {0} - start: {1}", exitCode, title),
                            session,
                            customData
                        );
                    }

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
                RollbarReport.Error(exceptionString, session);
                return ActionResult.Failure;
            }
            output = outputBuilder.ToString();
            return ActionResult.Success;
        }

        /// <summary>
        /// FormatErrorOutput formats the output of a state tool command optimized for display in an error dialog
        /// </summary>
        /// <param name="cmdOutput">
        /// the output from a state tool command run with `--output=json`
        /// </param>
        public static string FormatErrorOutput(string cmdOutput)
        {
            return string.Join("\n", cmdOutput.Split('\x00').Select(blob =>
            {
                try
                {
                    var json = new JavaScriptSerializer();
                    var data = json.Deserialize<Dictionary<string, string>>(blob);
                    var error = data["Error"];
                    return error;
                }
                catch (Exception)
                {
                    return blob;
                }
            }).ToList());

        }
    }
}
