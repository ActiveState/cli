using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Text;

namespace StateDeploy
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult StateDeploy(Session session)
        {
            session.Log("Starting state deploy");

            StatusMessage(session, string.Format("Deploying project {0}...", session.CustomActionData["PROJECT_NAME"]));
            MessageResult incrementResult = IncrementProgressBar(session, 3);
            if (incrementResult == MessageResult.Cancel)
            {
                return ActionResult.UserExit;
            }
            string deployCmd = BuildDeployCmd(session);
            session.Log(string.Format("Executing deploy command: {0}", deployCmd));
            try
            {
                System.Diagnostics.ProcessStartInfo procStartInfo =
                    new System.Diagnostics.ProcessStartInfo("cmd", "/c " + deployCmd);

                // The following commands are needed to redirect the standard output.
                // This means that it will be redirected to the Process.StandardOutput StreamReader.

                // NOTE: Due to progress bar changes in the State Tool we can no longer redirect stdout
                // and strerr output. Once we have a non-interactive mode in the State Tool these lines
                // can be enabled
                // procStartInfo.RedirectStandardOutput = true;
                // procStartInfo.RedirectStandardError = true;

                procStartInfo.UseShellExecute = false;
                // Do not create the black window.
                procStartInfo.CreateNoWindow = true;

                System.Diagnostics.Process proc = new System.Diagnostics.Process();
                proc.StartInfo = procStartInfo;
                proc.Start();
                proc.WaitForExit();
                
                // NOTE: See comment above re: progress bar. Can enable these lines once State Tool
                // is updated
                // session.Log(string.Format("Standard output: {0}", proc.StandardOutput.ReadToEnd()));
                // session.Log(string.Format("Standard error: {0}", proc.StandardError.ReadToEnd()));
            }
            catch (Exception objException)
            {
                session.Log(string.Format("Caught exception: {0}", objException));
                return ActionResult.Failure;
            }

            return ActionResult.Success;
        }

        private static string BuildDeployCmd(Session session)
        {
            string installDir = session.CustomActionData["INSTALLDIR"];
            string projectName = session.CustomActionData["PROJECT_NAME"];
            string isModify = session.CustomActionData["IS_MODIFY"];

            StringBuilder deployCMDBuilder = new StringBuilder("state deploy");
            if (isModify == "true")
            {
                deployCMDBuilder.Append(" --force");
            }

            // We quote the string here as Windows paths that contain spaces must be quoted.
            // We also account for a path ending with a slash and ensure that the quote character
            // isn't preserved.
            deployCMDBuilder.AppendFormat(" {0} --path=\"{1}\\\"", projectName, @installDir);

            return deployCMDBuilder.ToString();
        }

        internal static void StatusMessage(Session session, string status)
        {
            Record record = new Record(3);
            record[1] = "callAddProgressInfo";
            record[2] = status;
            record[3] = "Incrementing tick [1] of [2]";

            session.Message(InstallMessage.ActionStart, record);
        }

        public static MessageResult IncrementProgressBar(Session session, int progressPercentage)
        {
            var record = new Record(3);
            record[1] = 2; // "ProgressReport" message 
            record[2] = progressPercentage.ToString(); // ticks to increment 
            record[3] = 0; // ignore 
            return session.Message(InstallMessage.Progress, record);
        }
    }
}
