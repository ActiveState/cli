using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Text;
using System.IO;
using System.Net;
using System.Collections.ObjectModel;
using System.Windows.Forms;
using System.Linq;
using System.Web.Script.Serialization;
using System.Collections.Generic;
using Newtonsoft.Json;
using System.Security.Cryptography;
using System.IO.Compression;
using ActiveState;
using Microsoft.Win32;

namespace StateDeploy
{
    public class CustomActions
    {
        private const string networkErrorKey = "NetworkError";
        private const string networkErrorMessageKey = "NetworkErrorMessage";
        private const string sessionIDKey = "SessionID";

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
            // Registry info for network errors
            // This custom action runs as administrator so we have to specifically set
            // the registry key for the user using their SID in order for the value to
            // be available in later immediate custom actions
            string registryKey = string.Format("HKEY_USERS\\{0}\\SOFTWARE\\ActiveState\\{1}", session.CustomActionData["USERSID"], session.CustomActionData["PRODUCT_NAME"]);
            RegistryValueKind registryEntryDataType = RegistryValueKind.String;
            try
            {
                Registry.SetValue(registryKey, networkErrorKey, "false", registryEntryDataType);
                Registry.SetValue(registryKey, networkErrorMessageKey, "", registryEntryDataType);
            } catch (Exception e)
            {
                string msg = string.Format("Could not delete network error registry keys. Exception: {0}", e.ToString());
                session.Log(msg);
                RollbarReport.Error(msg, session);
            }

            var paths = GetPaths();
            string stateURL = "https://s3.ca-central-1.amazonaws.com/cli-update/update/state/unstable/";
            string jsonURL = stateURL + paths.JsonDescription;
            string timeStamp = DateTime.Now.ToFileTime().ToString();
            string tempDir = Path.Combine(Path.GetTempPath(), timeStamp);
            string stateToolInstallDir = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "ActiveState", "bin");
            stateToolPath = Path.Combine(stateToolInstallDir, "state.exe");

            if (File.Exists(stateToolPath))
            {
                session.Log("Using existing State Tool executable at install path");
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
                SetNetworkErrorDetails(session, registryKey, e);
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
                SetNetworkErrorDetails(session, registryKey, e);
                return ActionResult.Failure;
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
                    File.WriteAllText(installFilePath, contents);
                }
                catch (Exception e)
                {
                    string msg = string.Format("Could not write install file at path: {0}, encountered exception: {1}", output, e.ToString());
                    session.Log(msg);
                    RollbarReport.Error(msg, session);
                }
            }

            session.Log("Updating PATH environment variable");
            string oldPath = Environment.GetEnvironmentVariable("PATH", EnvironmentVariableTarget.Machine);
            if (oldPath.Contains(stateToolInstallDir))
            {
                session.Log("State tool installation already on PATH");
                return ActionResult.Success;
            }

            var newPath = string.Format("{0};{1}", stateToolInstallDir, oldPath);
            session.Log(string.Format("updating PATH to {0}", newPath));
            try
            {
                Environment.SetEnvironmentVariable("PATH", newPath, EnvironmentVariableTarget.Machine);
            }
            catch (Exception e)
            {
                string msg = string.Format("Could not update PATH. Attempted to set path to: {0}, encountered exception: {1}", newPath, e.ToString());
                session.Log(msg);
                RollbarReport.Critical(msg, session);
                return ActionResult.Failure;
            }

            return ActionResult.Success;

        }

        private static void SetNetworkErrorDetails(Session session, string registryKey, Exception e)
        {
            RegistryValueKind registryEntryDataType = RegistryValueKind.String;
            try
            {
                Registry.SetValue(registryKey, networkErrorKey, "true", registryEntryDataType);
                Registry.SetValue(registryKey, networkErrorMessageKey, e.Message, registryEntryDataType);
            }
            catch (Exception registryException)
            {
                string registryExceptionMsg = string.Format("Could not set network error registry values. Exception: {0}", registryException.ToString());
                session.Log(registryExceptionMsg);
                RollbarReport.Error(registryExceptionMsg, session);
            }
        }
        public static ActionResult InstallStateTool(Session session, string msiLogFileName, out string stateToolPath)
        {
            RollbarHelper.ConfigureRollbarSingleton(session.CustomActionData["MSI_VERSION"]);
            var productVersion = session.CustomActionData["PRODUCT_VERSION"];

            session.Log("Installing State Tool if necessary");
            if (session.CustomActionData["STATE_TOOL_INSTALLED"] == "true")
            {
                stateToolPath = session.CustomActionData["STATE_TOOL_PATH"];
                session.Log("State Tool is installed, no installation required");
                Status.ProgressBar.Increment(session, 1);
                TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "stage", "state-tool", "skipped", productVersion);

                return ActionResult.Success;
            }

            Status.ProgressBar.StatusMessage(session, "Installing State Tool...");
            Status.ProgressBar.Increment(session, 1);

            var ret = _installStateTool(session, out stateToolPath);
            if (ret == ActionResult.Success)
            {
                TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "stage", "state-tool", "success", productVersion);
            }
            else if (ret == ActionResult.Failure)
            {
                TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "stage", "state-tool", "failure", productVersion);
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
            ActionResult runResult = ActiveState.Command.Run(session, stateToolPath, authCmd, out output);
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
            var msiLogFileName = session.CustomActionData["MsiLogFileLocation"];
            var uiLevel = session.CustomActionData["UI_LEVEL"];
            var productVersion = session.CustomActionData["PRODUCT_VERSION"];

            if (uiLevel == "2" /* no ui */ || uiLevel == "3" /* basic ui */)
            {
                // we have to send the start event, because it has not triggered before
                reportStartEvent(session, msiLogFileName, productVersion, uiLevel);
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
            ActionResult res = InstallStateTool(session, msiLogFileName, out stateToolPath);
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

                    Status.ProgressBar.Increment(session, 1);
                    Status.ProgressBar.StatusMessage(session, seq.Description);

                    string output;
                    var runResult = ActiveState.Command.Run(session, stateToolPath, deployCmd, out output);
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
                        TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "stage", "artifacts", "failure", productVersion);

                        return runResult;
                    }
                }
                TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "stage", "artifacts", "success", productVersion);
            }
            catch (Exception objException)
            {
                string msg = string.Format("Caught exception: {0}", objException);
                session.Log(msg);
                RollbarReport.Critical(msg, session);
                return ActionResult.Failure;
            }

            Status.ProgressBar.Increment(session, 1);
            return ActionResult.Success;

        }


        [CustomAction]
        public static ActionResult StateDeploy(Session session)
        {
            ActiveState.RollbarHelper.ConfigureRollbarSingleton(session.CustomActionData["MSI_VERSION"]);
            return run(session);
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
            if (subCommand == "install" && commitID != "latest")
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

        public static void reportStartEvent(Session session, string msiLogFileName, string productVersion, string uiLevel)
        {
            session.Log("sending MSI start - event");
            TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "stage", "started", uiLevel, productVersion);
        }

        [CustomAction]
        public static ActionResult GAReportStart(Session session)
        {
            reportStartEvent(session, session["MsiLogFileLocation"], session["ProductVersion"], session["UILevel"]);
            return ActionResult.Success;
        }

        [CustomAction]
        public static ActionResult GAReportFailure(Session session)
        {
            var msiLogFileName = session["MsiLogFileLocation"];
            session.Log("sending event about MSI failure");

            TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "stage", "finished", "failure", session["ProductVersion"]);
            return ActionResult.Success;
        }

        [CustomAction]
        public static ActionResult GAReportSuccess(Session session)
        {
            var msiLogFileName = session["MsiLogFileLocation"];
            session.Log("sending event about MSI success");

            TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "stage", "finished", "success", session["ProductVersion"]);
            return ActionResult.Success;
        }

        /// <summary>
        /// Reports a user cancellation event to google analytics
        /// </summary>
        [CustomAction]
        public static ActionResult GAReportUserExit(Session session)
        {
            var msiLogFileName = session["MsiLogFileLocation"];

            session.Log("sending user exit event");
            TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "stage", "finished", "cancelled", session["ProductVersion"]);
            return ActionResult.Success;
        }

        /// <summary>
        /// Reports a user network error event to google analytics
        /// </summary>
        [CustomAction]
        public static ActionResult GAReportUserNetwork(Session session)
        {
            var msiLogFileName = session["MsiLogFileLocation"];

            session.Log("sending user network error event");
            TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "stage", "finished", "user_network", session["ProductVersion"]);
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
        public static ActionResult SetNetworkErrorProperties(Session session)
        {
            session.Log("Begin SetNetworkErrorProperties");

            // Get the registry values set on error in the _installStateTool function
            // Do not fail if we cannot get the values, simply present the fatal custom
            // error dialog without any mention of network errors
            string registryKey = string.Format("SOFTWARE\\ActiveState\\{0}", session["ProductName"]);
            RegistryKey productKey = Registry.CurrentUser.CreateSubKey(registryKey);
            try
            {
                Object networkError = productKey.GetValue(networkErrorKey);
                Object networkErrorMessage = productKey.GetValue(networkErrorMessageKey);
                session["NETWORK_ERROR"] = networkError as string;
                session["NETWORK_ERROR_MESSAGE"] = networkErrorMessage as string;
            } catch (Exception e)
            {
                string msg = string.Format("Could not read network error registry keys. Exception: {0}", e.ToString());
                session.Log(msg);
                RollbarReport.Error(msg, session);
            }

            if (session["NETWORK_ERROR"] == "true") {
                session.DoAction("GAReportUserNetwork");
            } else
            {
                session.DoAction("GAReportFailure");
            }

            session.DoAction("CustomFatalError");
            return ActionResult.Success;
        }
    }
}
