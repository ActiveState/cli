using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Text;
using System.Diagnostics;

namespace StateDeploy
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult StateDeploy(Session session)
        {
            session.Log("Starting state deploy");

            Status.ProgressBar.StatusMessage(session, string.Format("Deploying project {0}...", session.CustomActionData["PROJECT_NAME"]));
            MessageResult incrementResult = Status.ProgressBar.Increment(session, 3);
            if (incrementResult == MessageResult.Cancel)
            {
                return ActionResult.UserExit;
            }

            string deployCmd = BuildDeployCmd(session);
            session.Log(string.Format("Executing deploy command: {0}", deployCmd));
            try
            {
                ProcessStartInfo procStartInfo = new ProcessStartInfo("cmd", "/c " + deployCmd);

                // The following commands are needed to redirect the standard output.
                // This means that it will be redirected to the Process.StandardOutput StreamReader.

                // NOTE: Due to progress bar changes in the State Tool we can no longer redirect stdout
                // and strerr output. Once we have a non-interactive mode in the State Tool these lines
                // can be enabled
                procStartInfo.RedirectStandardOutput = true;
                //procStartInfo.RedirectStandardError = true;

                procStartInfo.UseShellExecute = false;
                // Do not create the black window.
                procStartInfo.CreateNoWindow = true;

                Process proc = new Process();
                proc.StartInfo = procStartInfo;
                proc.Start();

                while (!proc.HasExited)
                {
                    try
                    {
                        Status.ProgressBar.Increment(session, 0);
                        System.Threading.Thread.Sleep(200);
                    } catch (InstallCanceledException)
                    {
                        session.Log("Caught install canceled exception");

                        ActiveState.Process.KillProcessAndChildren(proc.Id);

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
                }
                
                // NOTE: See comment above re: progress bar. Can enable these lines once State Tool
                // is updated
                session.Log(string.Format("Standard output: {0}", proc.StandardOutput.ReadToEnd()));
                //session.Log(string.Format("Standard error: {0}", proc.StandardError.ReadToEnd()));

                if (proc.ExitCode != 0)
                {
                    session.Log(string.Format("Process exited with code: {0}", proc.ExitCode));
                    return ActionResult.Failure;
                }
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

            StringBuilder deployCMDBuilder = new StringBuilder(session.CustomActionData["STATE_TOOL_PATH"] + " deploy");
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
