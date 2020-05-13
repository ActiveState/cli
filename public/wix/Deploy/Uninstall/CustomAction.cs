using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using Microsoft.Deployment.WindowsInstaller;

namespace Uninstall
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult Uninstall(Session session)
        {
            session.Log("Begin uninstallation");

            string installDir = session["INSTALLDIR"];
            try
            {
                Directory.Delete(installDir, true);
            } 
            catch (IOException e)
            {
                session.Log(string.Format("Could not delete install directory, got error: {0}", e.ToString()));
                return ActionResult.Failure;
            }

            string pathEnv = Environment.GetEnvironmentVariable("PATH", EnvironmentVariableTarget.User);
            session.Log(string.Format("PATH: {0}", pathEnv));
            string[] paths = pathEnv.Split(';');

            List<string> cleanPath = new List<string>();
            foreach (var path in paths)
            {
                if (path.StartsWith(installDir))
                {
                    continue;
                }
                cleanPath.Add(path);
            }

            Environment.SetEnvironmentVariable("PATH", string.Join(";", cleanPath), EnvironmentVariableTarget.User);
            Environment.SetEnvironmentVariable("PATH_OLD", null);

            return ActionResult.Success;
        }
    }
}
