using System;
using System.Collections.Generic;
using System.Text;
using Microsoft.Deployment.WindowsInstaller;

namespace StateDeploy
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult StateDeploy(Session session)
        {
            session.Log("Begin state deploy");

            string projectName = session["PROJECT_NAME"];
            string targetDir = session["INSTALLDIR"];

            try
            {
                System.Diagnostics.ProcessStartInfo procStartInfo =
                    new System.Diagnostics.ProcessStartInfo("cmd", "/c " + string.Format("state deploy --path {0} {1}", targetDir, projectName));

                // The following commands are needed to redirect the standard output.
                // This means that it will be redirected to the Process.StandardOutput StreamReader.
                procStartInfo.RedirectStandardOutput = true;
                procStartInfo.UseShellExecute = false;
                // Do not create the black window.
                procStartInfo.CreateNoWindow = true;

                System.Diagnostics.Process proc = new System.Diagnostics.Process();
                proc.StartInfo = procStartInfo;
                proc.Start();
            }
            catch (Exception objException)
            {
                session.Log(string.Format("Caught exception: {0}", objException));
                ActionResult.Failure;
            }

            return ActionResult.Success;
        }
    }
}
