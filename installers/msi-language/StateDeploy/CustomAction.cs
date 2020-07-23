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
using System.Web.Script.Serialization;
using System.Collections.Generic;

namespace StateDeploy
{

    enum Shell
    {
        Cmd,
        Powershell,
    }

    public class CustomActions
    {
        public static ActionResult InstallStateTool(Session session, out string stateToolPath)
        {
            ActiveState.RollbarHelper.ConfigureRollbarSingleton();

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
                ActiveState.RollbarHelper.Report(string.Format("Encoutered exception downloading file: {0}", e.ToString()));
                return ActionResult.Failure;
            }

            string installCmd = string.Format("Set-PSDebug -trace 2; Set-ExecutionPolicy Unrestricted -Scope Process; \"{0}\" -n -t \"{1}\"", scriptPath, installPath);
            session.Log(string.Format("Running install command: {0}", installCmd));

            string output;
            ActionResult result = ActiveState.Command.Run(session, installCmd, ActiveState.Shell.Powershell, out output);
            if (result.Equals(ActionResult.UserExit))
            {
                // Catch cancel and return
                return result;
            }
            else if (result.Equals(ActionResult.Failure))
            {
                Record record = new Record();
                var errorOutput = FormatErrorOutput(output);
                record.FormatString = String.Format("state tool installation failed with error:\n{0}", errorOutput);

                MessageResult msgRes = session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                return result;
            }
            Status.ProgressBar.Increment(session, 1);

            stateToolPath = Path.Combine(installPath, "state.exe");
            return result;
        }

        private static ActionResult Login(Session session, string stateToolPath)
        {
            string username = session.CustomActionData["AS_USERNAME"];
            string password = session.CustomActionData["AS_PASSWORD"];
            string totp = session.CustomActionData["AS_TOTP"];

            if (username == "" && password == "" && totp == "")
            {
                session.Log("No login information provided, not executing login");
                return ActionResult.Success;
            }

            string authCmd;
            if (totp != "")
            {
                session.Log("Attempting to log in with TOTP token");
                authCmd = stateToolPath + " auth" + " --totp " + totp;
            } else
            {
                session.Log(string.Format("Attempting to login as user: {0}", username));
                authCmd = stateToolPath + " auth" + " --username " + username + " --password " + password;
            }

            string output;
            Status.ProgressBar.StatusMessage(session, "Authenticating...");
            ActionResult runResult = ActiveState.Command.Run(session, authCmd, ActiveState.Shell.Cmd, out output);
            if (runResult.Equals(ActionResult.UserExit))
            {
                // Catch cancel and return
                return runResult;
            }
            else if (runResult == ActionResult.Failure)
            {
                Record record = new Record();
                session.Log(string.Format("Output: {0}", output));
                var errorOutput = FormatErrorOutput(output);
                record.FormatString = string.Format("Failed with error:\n{0}", errorOutput);

                session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                return runResult;
            }
            // The auth command did not fail but the username we expected is not present in the output meaning
            // another user is logged into the State Tool 
            else if (!output.Contains(username))
            {
                Record record = new Record();
                var errorOutput = string.Format("Could not log in as {0}, currently logged in as another user. To correct this please start a command prompt and execute {1} auth logout and try again", username, stateToolPath);
                record.FormatString = string.Format("Failed with error:\n{0}", errorOutput);

                session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                return ActionResult.Failure;
            }
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
            ActiveState.RollbarHelper.ConfigureRollbarSingleton();

            string stateToolPath;
            ActionResult res = InstallStateTool(session, out stateToolPath);
            if (res != ActionResult.Success) {
                return res;
            }
            session.Log("Starting state deploy with state tool at " + stateToolPath);

            res = Login(session, stateToolPath);
            if (res.Equals(ActionResult.Failure))
            {
                return res;
            }

            Status.ProgressBar.StatusMessage(session, string.Format("Deploying project {0}...", session.CustomActionData["PROJECT_OWNER_AND_NAME"]));
            Status.ProgressBar.StatusMessage(session, string.Format("Preparing deployment of {0}...", session.CustomActionData["PROJECT_OWNER_AND_NAME"]));

            var sequence = new ReadOnlyCollection<InstallSequenceElement>(
                new[]
                {
                    new InstallSequenceElement("install", string.Format("Installing {0}", session.CustomActionData["PROJECT_OWNER_AND_NAME"])),
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
                    var runResult = ActiveState.Command.Run(session, deployCmd, ActiveState.Shell.Cmd, out output);
                    if (runResult.Equals(ActionResult.UserExit))
                    {
                        // Catch cancel and return
                        return runResult;
                    }
                    else if (runResult == ActionResult.Failure)
                    {
                        Record record = new Record();
                        var errorOutput = FormatErrorOutput(output);
                        record.FormatString = String.Format("{0} failed with error:\n{1}", seq.Description, errorOutput);

                        MessageResult msgRes = session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                        return runResult;
                    }
                }
            }
            catch (Exception objException)
            {
                session.Log(string.Format("Caught exception: {0}", objException));
                ActiveState.RollbarHelper.Report(string.Format("Caught exception: {0}", objException));
                return ActionResult.Failure;
            }

            Status.ProgressBar.Increment(session, 1);
            return ActionResult.Success;
        }

        /// <summary>
        /// FormatErrorOutput formats the output of a state tool command optimized for display in an error dialog
        /// </summary>
        /// <param name="cmdOutput">
        /// the output from a state tool command run with `--output=json`
        /// </param>
        private static string FormatErrorOutput(string cmdOutput)
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

        private static string BuildDeployCmd(Session session, string subCommand, string stateToolPath)
        {
            string installDir = session.CustomActionData["INSTALLDIR"];
            string projectName = session.CustomActionData["PROJECT_OWNER_AND_NAME"];
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
