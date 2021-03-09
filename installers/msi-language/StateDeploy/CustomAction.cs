using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Text;
using System.IO;
using System.Net;
using System.Collections.ObjectModel;
using System.Windows.Forms;
using System.Linq;
using Newtonsoft.Json;
using System.Security.Cryptography;
using System.IO.Compression;
using System.Windows.Forms.VisualStyles;
using ActiveState;
using Microsoft.Win32;
using System.Threading.Tasks;
using System.Threading;

namespace StateDeploy
{
    public class CustomActions
    {
        private struct StateToolPaths
        {
            public string JsonDescription;
            public string ZipFile;
            public string ExeFile;
        }

        private class VersionInfo
        {
            public string version = "";
            public string sha256v2 = "";
        }

        private static bool is64Bit()
        {
            return System.Environment.Is64BitOperatingSystem;
        }

        private static StateToolPaths GetPaths()
        {
            StateToolPaths paths;
            if (is64Bit())
            {
                paths.JsonDescription = "windows-amd64.json";
                paths.ZipFile = "windows-amd64.zip";
                paths.ExeFile = "windows-amd64.exe";
            }
            else
            {
                paths.JsonDescription = "windows-386.json";
                paths.ZipFile = "windows-386.zip";
                paths.ExeFile = "windows-386.exe";
            }
            return paths;
        }

        private static ActionResult _installStateTool(Session session, out string stateToolPath)
        {
            Error.ResetErrorDetails(session);

            var paths = GetPaths();
            string stateURL = "https://state-tool.s3.amazonaws.com/update/state/release/";
            string jsonURL = stateURL + paths.JsonDescription;
            string timeStamp = DateTime.Now.ToFileTime().ToString();
            string tempDir = Path.Combine(Path.GetTempPath(), timeStamp);
            string stateToolInstallDir = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "ActiveState", "bin");
            stateToolPath = Path.Combine(stateToolInstallDir, "state.exe");

            if (File.Exists(stateToolPath))
            {
                session.Log("Using existing State Tool executable at install path");
                Status.ProgressBar.Increment(session, 200);
                return ActionResult.Success;
            }

            session.Log(string.Format("Using temp path: {0}", tempDir));
            try
            {
                Directory.CreateDirectory(tempDir);
            }
            catch (Exception e)
            {
                string msg = string.Format("Could not create temp directory at: {0}, encountered exception: {1}", tempDir, e.ToString());
                session.Log(msg);
                RollbarReport.Critical(msg, session);
                return ActionResult.Failure;
            }

            ServicePointManager.SecurityProtocol |= SecurityProtocolType.Tls11 | SecurityProtocolType.Tls12;

            string versionInfoString = "unset";
            session.Log(string.Format("Downloading JSON from URL: {0}", jsonURL));
            try
            {
                RetryHelper.RetryOnException(session, 3, TimeSpan.FromSeconds(2), () =>
                {
                    var client = new WebClient();
                    versionInfoString = client.DownloadString(jsonURL);
                });
            }
            catch (WebException e)
            {
                string msg = string.Format("Encountered exception downloading state tool json info file: {0}", e.ToString());
                session.Log(msg);
                new NetworkError().SetDetails(session, e.Message);
                return ActionResult.Failure;
            }

            VersionInfo info;
            try
            {
                info = JsonConvert.DeserializeObject<VersionInfo>(versionInfoString);
            }
            catch (Exception e)
            {
                string msg = string.Format("Could not deserialize version info. Version info string {0}, exception {1}", versionInfoString, e.ToString());
                session.Log(msg);
                RollbarReport.Critical(msg, session);
                return ActionResult.Failure;
            }

            string zipPath = Path.Combine(tempDir, paths.ZipFile);
            string zipURL = stateURL + info.version + "/" + paths.ZipFile;
            session.Log(string.Format("Downloading zip file from URL: {0}", zipURL));
            Status.ProgressBar.StatusMessage(session, "Downloading State Tool...");

            var tokenSource = new CancellationTokenSource();
            var token = tokenSource.Token;

            Task incrementTask = Task.Run(() =>
            {
                incrementProgressBar(session, 50, token);
            });

            Task<ActionResult> downloadTask = Task.Run(() =>
            {
                try
                {
                    RetryHelper.RetryOnException(session, 3, TimeSpan.FromSeconds(2), () =>
                    {
                        var client = new WebClient();
                        client.DownloadFile(zipURL, zipPath);
                    });
                }
                catch (WebException e)
                {
                    string msg = string.Format("Encountered exception downloading state tool zip file. URL to zip file: {0}, path to save zip file to: {1}, exception: {2}", zipURL, zipPath, e.ToString());
                    session.Log(msg);
                    new NetworkError().SetDetails(session, e.Message);
                    return ActionResult.Failure;
                }

                return ActionResult.Success;
            });

            ActionResult result = downloadTask.Result;
            tokenSource.Cancel();
            incrementTask.Wait();
            if (result.Equals(ActionResult.Failure))
            {
                return result;
            }


            SHA256 sha = SHA256.Create();
            FileStream fInfo = File.OpenRead(zipPath);
            string zipHash = BitConverter.ToString(sha.ComputeHash(fInfo)).Replace("-", string.Empty).ToLower();
            if (zipHash != info.sha256v2)
            {
                string msg = string.Format("SHA256 checksum did not match, expected: {0} actual: {1}", info.sha256v2, zipHash.ToString());
                session.Log(msg);
                RollbarReport.Critical(msg, session);
                return ActionResult.Failure;
            }

            Status.ProgressBar.StatusMessage(session, "Extracting State Tool executable...");
            Status.ProgressBar.Increment(session, 50);
            try
            {
                ZipFile.ExtractToDirectory(zipPath, tempDir);
            }
            catch (Exception e)
            {
                string msg = string.Format("Could not extract State Tool, encountered exception. Path to zip file: {0}, path to temp directory: {1}, exception {2})", zipPath, tempDir, e);
                session.Log(msg);
                RollbarReport.Critical(msg, session);
                return ActionResult.Failure;
            }

            try
            {
                Directory.CreateDirectory(stateToolInstallDir);
            }
            catch (Exception e)
            {
                string msg = string.Format("Could not create State Tool install directory at: {0}, encountered exception: {1}", stateToolInstallDir, e.ToString());
                session.Log(msg);
                RollbarReport.Critical(msg, session);
                return ActionResult.Failure;
            }

            try
            {
                File.Move(Path.Combine(tempDir, paths.ExeFile), stateToolPath);
            }
            catch (Exception e)
            {
                string msg = string.Format("Could not move State Tool executable to: {0}, encountered exception: {1}", stateToolPath, e);
                session.Log(msg);
                RollbarReport.Critical(msg, session);
                return ActionResult.Failure;
            }


            string configDirCmd = " export" + " config" + " --filter=dir";
            string output;
            ActionResult runResult = ActiveState.Command.Run(session, stateToolPath, configDirCmd, out output);
            session.Log("Writing install file...");
            // We do not fail the installation if writing the installsource.txt file fails
            if (runResult.Equals(ActionResult.Failure))
            {
                string msg = string.Format("Could not get config directory from State Tool");
                session.Log(msg);
                RollbarReport.Error(msg, session);
            }
            else
            {
                string contents = "msi-ui";
                if (session.CustomActionData["UI_LEVEL"] == "2")
                {
                    contents = "msi-silent";
                }
                try
                {
                    string installFilePath = Path.Combine(output.Trim(), "installsource.txt");
                    File.WriteAllText(installFilePath, contents, Encoding.ASCII);
                }
                catch (Exception e)
                {
                    string msg = string.Format("Could not write install file at path: {0}, encountered exception: {1}", output, e.ToString());
                    session.Log(msg);
                    RollbarReport.Error(msg, session);
                }
            }

            session.Log("Updating PATH environment variable");
            Status.ProgressBar.Increment(session, 50);
            string oldPath = Environment.GetEnvironmentVariable("PATH", EnvironmentVariableTarget.Machine);
            if (oldPath.Contains(stateToolInstallDir))
            {
                session.Log("State tool installation already on PATH");
            }
            else
            {
                var newPath = string.Format("{0};{1}", stateToolInstallDir, oldPath);
                session.Log(string.Format("updating PATH to {0}", newPath));
                try
                {
                    Environment.SetEnvironmentVariable("PATH", newPath, EnvironmentVariableTarget.Machine);
                }
                catch (Exception e)
                {
                    string msg = string.Format("Could not update PATH. Encountered exception: {0}", e.Message);
                    session.Log(msg);
                    new SecurityError().SetDetails(session, msg);
                    return ActionResult.Failure;
                }
            }

            session.Log("Running prepare step...");
            string prepareCmd = " _prepare";
            string prepareOutput;
            ActionResult prepareRunResult = ActiveState.Command.Run(session, stateToolPath, prepareCmd, out prepareOutput);
            if (prepareRunResult.Equals(ActionResult.Failure))
            {
                string msg = string.Format("Preparing environment caused error: {0}", prepareOutput);
                session.Log(msg);
                RollbarReport.Critical(msg, session);

                Record record = new Record();
                var errorOutput = Command.FormatErrorOutput(prepareOutput);
                record.FormatString = msg;

                session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                return ActionResult.Failure;
            }
            else
            {
                session.Log(string.Format("Prepare Output: {0}", prepareOutput));
            }

            Status.ProgressBar.Increment(session, 50);
            return ActionResult.Success;
        }

        private static void incrementProgressBar(Session session, int limit, CancellationToken ct)
        {
            if (ct.IsCancellationRequested)
            {
                session.Log("Cancelling incrementProgressBar");
                return;
            }
            for (int i = 0; i <= limit; i++)
            {
                if (ct.IsCancellationRequested)
                {
                    session.Log("Cancelling incrementProgressBar");
                    return;
                }
                Status.ProgressBar.Increment(session, 1);
                Thread.Sleep(150);
            }
        }

        public static ActionResult InstallStateTool(Session session, out string stateToolPath)
        {
            RollbarHelper.ConfigureRollbarSingleton(session.CustomActionData["MSI_VERSION"]);

            session.Log("Installing State Tool if necessary");
            if (session.CustomActionData["STATE_TOOL_INSTALLED"] == "true")
            {
                stateToolPath = session.CustomActionData["STATE_TOOL_PATH"];
                session.Log("State Tool is installed, no installation required");
                Status.ProgressBar.Increment(session, 250);
                TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "state-tool", "skipped");

                return ActionResult.Success;
            }

            Status.ProgressBar.StatusMessage(session, "Installing State Tool...");
            Status.ProgressBar.Increment(session, 50);

            var ret = _installStateTool(session, out stateToolPath);
            if (ret == ActionResult.Success)
            {
                TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "state-tool", "success");
            }
            else if (ret == ActionResult.Failure)
            {
                TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "state-tool", "failure");
            }
            return ret;
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
                authCmd = " auth" + " --totp " + totp;
            }
            else
            {
                session.Log(string.Format("Attempting to login as user: {0}", username));
                authCmd = " auth" + " --username " + username + " --password " + password;
            }

            string output;
            Status.ProgressBar.StatusMessage(session, "Authenticating...");
            ActionResult runResult = ActiveState.Command.RunAuthCommand(session, stateToolPath, authCmd, out output);
            if (runResult.Equals(ActionResult.UserExit))
            {
                // Catch cancel and return
                return runResult;
            }
            else if (runResult == ActionResult.Failure)
            {
                Record record = new Record();
                session.Log(string.Format("Output: {0}", output));
                var errorOutput = Command.FormatErrorOutput(output);
                record.FormatString = string.Format("Platform login failed with error:\n{0}", errorOutput);

                session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                return runResult;
            }
            // The auth command did not fail but the username we expected is not present in the output meaning
            // another user is logged into the State Tool
            else if (!output.Contains(username))
            {
                Record record = new Record();
                var errorOutput = string.Format("Could not log in as {0}, currently logged in as another user. To correct this please start a command prompt and execute `{1} auth logout` and try again", username, stateToolPath);
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

        private static ActionResult run(Session session)
        {
            var uiLevel = session.CustomActionData["UI_LEVEL"];

            if (uiLevel == "2" /* no ui */ || uiLevel == "3" /* basic ui */)
            {
                // we have to send the start event, because it has not triggered before
                reportStartEvent(session, uiLevel);
            }

            if (!Environment.Is64BitOperatingSystem)
            {
                Record record = new Record();
                record.FormatString = "This installer cannot be run on a 32-bit operating system";

                RollbarReport.Critical(record.FormatString, session);
                session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                return ActionResult.Failure;
            }

            string stateToolPath;
            ActionResult res = InstallStateTool(session, out stateToolPath);
            if (res != ActionResult.Success)
            {
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
                    string deployCmd = BuildDeployCmd(session, seq.SubCommand);
                    session.Log(string.Format("Executing deploy command: {0}", deployCmd));

                    Status.ProgressBar.StatusMessage(session, seq.Description);
                    string output;
                    var runResult = ActiveState.Command.RunWithProgress(session, stateToolPath, deployCmd, 200, out output);
                    if (runResult.Equals(ActionResult.UserExit))
                    {
                        // Catch cancel and return
                        return runResult;
                    }
                    else if (runResult == ActionResult.Failure)
                    {
                        Record record = new Record();
                        var errorOutput = Command.FormatErrorOutput(output);
                        record.FormatString = String.Format("{0} failed with error:\n{1}", seq.Description, errorOutput);

                        MessageResult msgRes = session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                        TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "artifacts", "failure");

                        return runResult;
                    }
                }
                TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "artifacts", "success");
            }
            catch (Exception objException)
            {
                string msg = string.Format("Caught exception: {0}", objException);
                session.Log(msg);
                RollbarReport.Critical(msg, session);
                return ActionResult.Failure;
            }

            Status.ProgressBar.Increment(session, 100);
            return ActionResult.Success;

        }


        [CustomAction]
        public static ActionResult StateDeploy(Session session)
        {
            ActiveState.RollbarHelper.ConfigureRollbarSingleton(session.CustomActionData["MSI_VERSION"]);
            return run(session);
        }

        private static string BuildDeployCmd(Session session, string subCommand)
        {
            string installDir = session.CustomActionData["INSTALLDIR"];
            string projectName = session.CustomActionData["PROJECT_OWNER_AND_NAME"];
            string isModify = session.CustomActionData["IS_MODIFY"];
            string commitID = session.CustomActionData["COMMIT_ID"];

            StringBuilder deployCMDBuilder = new StringBuilder(String.Format("deploy {0}", subCommand));
            if (isModify == "true" && subCommand == "symlink")
            {
                deployCMDBuilder.Append(" --force");
            }
            // Add commitID if requested
            if (commitID != "latest")
            {
                projectName += "#" + commitID;
            }
            deployCMDBuilder.Append(" --output json");

            // We quote the string here as Windows paths that contain spaces must be quoted.
            // We also account for a path ending with a slash and ensure that the quote character
            // isn't preserved.
            deployCMDBuilder.AppendFormat(" {0} --path=\"{1}\\\"", projectName, installDir);

            return deployCMDBuilder.ToString();
        }

        /* The following custom actions are added to this project (and not to a project
         * with a more appropriate name) in hope that the TrackerSingleton ca be re-used between
         * all custom actions.
         */

        public static void reportStartEvent(Session session, string uiLevel)
        {
            session.Log("sending MSI start - event");
            TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "started", uiLevel);
        }

        [CustomAction]
        public static ActionResult GAReportStart(Session session)
        {
            reportStartEvent(session, session["UILevel"]);
            return ActionResult.Success;
        }

        [CustomAction]
        public static ActionResult GAReportFailure(Session session)
        {
            session.Log("sending event about MSI failure");

            TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "finished", "failure");
            return ActionResult.Success;
        }

        [CustomAction]
        public static ActionResult GAReportSuccess(Session session)
        {
            session.Log("sending event about MSI success");

            TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "finished", "success");
            return ActionResult.Success;
        }

        /// <summary>
        /// Reports a user cancellation event to google analytics
        /// </summary>
        [CustomAction]
        public static ActionResult GAReportUserExit(Session session)
        {
            session.Log("sending user exit event");
            TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "finished", "cancelled");
            TrackerSingleton.Instance.TrackEventSynchronously(session, "exits", session["LAST_DIALOG"], "cancelled");
            return ActionResult.Success;
        }

        /// <summary>
        /// Reports a user network error event to google analytics
        /// </summary>
        [CustomAction]
        public static ActionResult GAReportUserNetwork(Session session)
        {
            session.Log("sending user network error event");
            TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "finished", "user_network");
            TrackerSingleton.Instance.TrackEventSynchronously(session, "exits", session["LAST_DIALOG"], "user_network");
            return ActionResult.Success;
        }

        /// <summary>
        /// Reports a user network error event to google analytics
        /// </summary>
        [CustomAction]
        public static ActionResult GAReportUserSecurity(Session session)
        {
            session.Log("sending antivirus error event");
            TrackerSingleton.Instance.TrackEventSynchronously(session, "stage", "finished", "user_security");
            TrackerSingleton.Instance.TrackEventSynchronously(session, "exits", session["LAST_DIALOG"], "user_security");

            return ActionResult.Success;
        }

        [CustomAction]
        public static ActionResult ValidateInstallFolder(Session session)
        {
            var installFolder = session["INSTALLDIR"];
            session.Log("Checking folder {0}", installFolder);

            session["VALIDATE_FOLDER_CLEAN"] = "0";
            if (!Directory.Exists(installFolder))
            {
                session.Log("Folder {0} does not exist.  Let's proceed.", installFolder);
                session["VALIDATE_FOLDER_CLEAN"] = "1";
                return ActionResult.Success;
            }

            if (Directory.EnumerateFileSystemEntries(installFolder).Any())
            {
                session.Log("Selected installation folder {0} exists and is not empty.", installFolder);
                return ActionResult.Success;
            };

            session.Log("Selected installation folder {0} exists, but is empty.  All good.", installFolder);
            session["VALIDATE_FOLDER_CLEAN"] = "1";
            return ActionResult.Success;

        }

        [CustomAction]
        public static ActionResult CustomOnError(Session session)
        {
            session.Log("Begin SetError");

            // Get the registry values set on error in the _installStateTool function
            // Do not fail if we cannot get the values, simply present the fatal custom
            // error dialog without any mention of network errors
            string registryKey = string.Format("SOFTWARE\\ActiveState\\{0}", session["ProductName"]);
            RegistryKey productKey = Registry.CurrentUser.CreateSubKey(registryKey);
            try
            {
                Object errorType = productKey.GetValue(Error.TypeRegistryKey);
                Object errorMessage = productKey.GetValue(Error.MessageRegistryKey);
                session.Log("errorType={0}, error_message={1}", errorType, errorMessage);
                session["ERROR"] = errorType as string;
                session["ERROR_MESSAGE"] = errorMessage as string;
            } catch (Exception e)
            {
                string msg = string.Format("Could not read network error registry keys. Exception: {0}", e.ToString());
                session.Log(msg);
                RollbarReport.Error(msg, session);
            }

            if (session["ERROR"] == new NetworkError().Type()) {
                session.Log("Network error type");
                session.DoAction("GAReportUserNetwork");
                session.DoAction("CustomNetworkError");
                RollbarReport.Error("user_network: " + session["ERROR_MESSAGE"], session);
            } else if (session["ERROR"] == new SecurityError().Type()) {
                session.Log("Path error type");
                session.DoAction("GAReportUserSecurity");
                session.DoAction("CustomSecurityError");
                RollbarReport.Error("user_security: " + session["ERROR_MESSAGE"], session);
            }
            else
            {
                session.Log("Default error type");
                session.DoAction("GAReportFailure");
                session.DoAction("CustomFatalError");
            }

            return ActionResult.Success;
        }


        [CustomAction]
        public static ActionResult CustomUserExit(Session session)
        {
            session.Log("Begin CustomUserExit");

            if (session["INSTALL_MODE"] == "Install") {
                session.DoAction("GAReportUserExit");
            }

            session.DoAction("CustomUserExitDialog");

            return ActionResult.Success;
        }
    }
}
