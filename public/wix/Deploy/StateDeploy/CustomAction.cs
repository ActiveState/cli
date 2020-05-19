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

            string deployCmd = BuildDeployCmd(session);
            session.Log(string.Format("Executing deploy command: {0}", deployCmd));
            try
            {
                System.Diagnostics.ProcessStartInfo procStartInfo =
                    new System.Diagnostics.ProcessStartInfo("cmd", "/c " + deployCmd);

                // The following commands are needed to redirect the standard output.
                // This means that it will be redirected to the Process.StandardOutput StreamReader.
                procStartInfo.RedirectStandardOutput = true;
                procStartInfo.RedirectStandardError = true;
                procStartInfo.UseShellExecute = false;
                // Do not create the black window.
                procStartInfo.CreateNoWindow = true;

                System.Diagnostics.Process proc = new System.Diagnostics.Process();
                proc.StartInfo = procStartInfo;
                proc.Start();
                session.Log(string.Format("Standard output: {0}", proc.StandardOutput.ReadToEnd()));
                session.Log(string.Format("Standard error: {0}", proc.StandardError.ReadToEnd()));
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
            string installDir = session["INSTALLDIR"];
            string projectName = session["PROJECT_NAME"];
            string isModify = session["IS_MODIFY"];

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
    }
}
