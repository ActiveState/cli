using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Text;
using System.Diagnostics;
using System.Threading;
using System.IO;
using System.Net;
using System.Collections.ObjectModel;
using System.Windows.Forms;
using System.Linq;

namespace StateDeploy
{
    public class CustomActions
    {
        public static ActionResult InstallStateTool(Session session, out string stateToolPath)
        {
            stateToolPath = "";
            session.Log("Installing State Tool if necessary");
            if (session.CustomActionData["STATE_TOOL_INSTALLED"] == "true")
            {
                stateToolPath = session.CustomActionData["STATE_TOOL_PATH"];
                session.Log("State Tool is installed, no installation required");
                Status.ProgressBar.Increment(session, 1);
                return ActionResult.Success;
            }

            string tempDir = Path.GetTempPath();
            string scriptPath = Path.Combine(tempDir, "install.ps1");
            string installPath = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "ActiveState", "bin");

            Status.ProgressBar.StatusMessage(session, "Installing State Tool...");

            ServicePointManager.SecurityProtocol |= SecurityProtocolType.Tls11 | SecurityProtocolType.Tls12;
            try
            {
                WebClient client = new WebClient();
                client.DownloadFile("https://platform.activestate.com/dl/cli/install.ps1", scriptPath);
            }
            catch (WebException e)
            {
                session.Log(string.Format("Encoutered exception downloading file: {0}", e.ToString()));
                return ActionResult.Failure;
            }

            string installCmd = string.Format("powershell \"{0} -n -t {1}\"", scriptPath, installPath);
            session.Log(string.Format("Running install command: {0}", installCmd));

            ActionResult result = RunCommand(session, installCmd);
            if (result.Equals(ActionResult.UserExit))
            {
                result = Uninstall.Remove.InstallDir(session, installPath);
                if (result.Equals(ActionResult.Failure))
                {
                    session.Log("Could not remove installation directory");
                    return ActionResult.Failure;
                }

                result = Uninstall.Remove.EnvironmentEntries(session, installPath);
                if (result.Equals(ActionResult.Failure))
                {
                    session.Log("Could not remove environment entries");
                    return ActionResult.Failure;
                }
                return ActionResult.UserExit;
            }
            Status.ProgressBar.Increment(session, 1);

            stateToolPath = Path.Combine(installPath, "state.exe");
            return result;
        }

        private static ActionResult RunCommand(Session session, string cmd)
        {
            return RunCommand(session, cmd, out _);
        }

        private static ActionResult RunCommand(Session session, string cmd, out string output)
        {
            var outputBuilder = new StringBuilder();
            try
            {
                ProcessStartInfo procStartInfo = new ProcessStartInfo("cmd.exe", "/c " + cmd);

                // The following commands are needed to redirect the standard output.
                // This means that it will be redirected to the Process.StandardOutput StreamReader.
                procStartInfo.RedirectStandardOutput = true;
                procStartInfo.RedirectStandardError = true;
                procStartInfo.UseShellExecute = false;
                // Do not create the black window.
                procStartInfo.CreateNoWindow = true;

                Process proc = new Process();
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
                    output = outputBuilder.ToString();
                    session.Log("returning due to return code - error");
                    session.Log("output = " + output);
                    return ActionResult.Failure;
                }
            }
            catch (Exception objException)
            {
                output = outputBuilder.ToString();
                session.Log(string.Format("Caught exception: {0}", objException));
                return ActionResult.Failure;
            }
            output = outputBuilder.ToString();
            return ActionResult.Success;
        }

        public struct InstallSequenceElement
        {
            public readonly string SubCommand;
            public readonly string Description;

            public InstallSequenceElement(string subCommand, string description)
            {
                this.SubCommand = subCommand;
                this.Description = description;
            }
        };


        [CustomAction]
        public static ActionResult StateDeploy(Session session)
        {
            string stateToolPath;
            var res = InstallStateTool(session, out stateToolPath);
            if (res != ActionResult.Success) {
                return res;
            }
            session.Log("Starting state deploy with state tool at " + stateToolPath);

            Status.ProgressBar.StatusMessage(session, string.Format("Deploying project {0}...", session.CustomActionData["PROJECT_NAME"]));
            MessageResult statusResult = Status.ProgressBar.StatusMessage(session, string.Format("Preparing deployment of {0}...", session.CustomActionData["PROJECT_NAME"]));
            if (statusResult == MessageResult.Cancel)
            {
                return ActionResult.UserExit;
            }

            var sequence = new ReadOnlyCollection<InstallSequenceElement>(
                new[]
                {
                    new InstallSequenceElement("install", string.Format("Installing {0}", session.CustomActionData["PROJECT_NAME"])),
                    new InstallSequenceElement("configure", "Updating system environment"),
                    new InstallSequenceElement("symlink", "Creating shortcut directory"),
                });

            try
            {
                foreach (var seq in sequence)
                {
                    string deployCmd = BuildDeployCmd(session, seq.SubCommand, stateToolPath);
                    session.Log(string.Format("Executing deploy command: {0}", deployCmd));

                    Status.ProgressBar.Increment(session, 1);
                    Status.ProgressBar.StatusMessage(session, seq.Description);

                    string output;
                    var runResult = RunCommand(session, deployCmd, out output);
                    if (runResult == ActionResult.UserExit)
                    {
                        ActionResult result = Uninstall.Remove.InstallDir(session, session.CustomActionData["INSTALLDIR"]);
                        if (result.Equals(ActionResult.Failure))
                        {
                            session.Log("Could not remove installation directory");
                            return ActionResult.Failure;
                        }

                        result = Uninstall.Remove.EnvironmentEntries(session, session.CustomActionData["INSTALLDIR"]);
                        if (result.Equals(ActionResult.Failure))
                        {
                            session.Log("Could not remove environment entries");
                            return ActionResult.Failure;
                        }
                        return ActionResult.UserExit;
                    }
                    else if (runResult == ActionResult.Failure)
                    {
                        Record record = new Record();
                        var sb = new StringBuilder();
                        var lastFiveLines = output.Split('\n').Reverse().Take(5).Reverse().ToList();
                        sb.AppendFormat("{0} failed with Error\n{1}", seq.Description, string.Join("\n", lastFiveLines));
                        record.FormatString = sb.ToString();

                        MessageResult msgRes = session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                        return runResult;
                    }
                }
            }
            catch (Exception objException)
            {
                session.Log(string.Format("Caught exception: {0}", objException));
                return ActionResult.Failure;
            }

            Status.ProgressBar.Increment(session, 1);
            return ActionResult.Success;
        }

        private static string BuildDeployCmd(Session session, string subCommand, string stateToolPath)
        {
            string installDir = session.CustomActionData["INSTALLDIR"];
            string projectName = session.CustomActionData["PROJECT_NAME"];
            string isModify = session.CustomActionData["IS_MODIFY"];

            StringBuilder deployCMDBuilder = new StringBuilder(stateToolPath + " deploy " + subCommand);
            if (isModify == "true")
            {
                deployCMDBuilder.Append(" --force");
            }

            deployCMDBuilder.Append(" --output json");

            // We quote the string here as Windows paths that contain spaces must be quoted.
            // We also account for a path ending with a slash and ensure that the quote character
            // isn't preserved.
            deployCMDBuilder.AppendFormat(" {0} --path=\"{1}\\\"", projectName, @installDir);

            return deployCMDBuilder.ToString();
        }
    }
}
