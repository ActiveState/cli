using System;
using System.Collections.Generic;
using System.IO;
using Microsoft.Deployment.WindowsInstaller;

namespace Uninstall
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult Uninstall(Session session)
        {
            session.Log("Begin uninstallation");

            string installDir = session.CustomActionData["REMEMBER"];

            ActionResult result = Remove.InstallDir(session, installDir);
            if (result.Equals(ActionResult.Failure))
            {
                session.Log("Could not remove installation directory");
                return ActionResult.Failure;
            }

            return Remove.EnvironmentEntries(session, installDir);
        }
    }
    public class Remove
    {
        public static ActionResult InstallDir(Session session, string dir)
        {
            session.Log("Begin removing install directory");

            if (Directory.Exists(dir))
            {
                try
                {
                    Directory.Delete(dir, true);
                }
                catch (IOException e)
                {
                    session.Log(string.Format("Could not delete install directory, got error: {0}", e.ToString()));
                    return ActionResult.Failure;
                }
            }

            return ActionResult.Success;
        }

        public static ActionResult EnvironmentEntries(Session session, string dir)
        {
            session.Log("Begin remvoing environment entries");
            string pathEnv = Environment.GetEnvironmentVariable("PATH", EnvironmentVariableTarget.Machine);
            if (pathEnv == null) {
              return ActionResult.Success;
            }
            string[] paths = pathEnv.Split(Path.PathSeparator);

            List<string> cleanPath = new List<string>();
            foreach (var path in paths)
            {
                if (path.StartsWith(dir))
                {
                    continue;
                }
                cleanPath.Add(path);
            }

            Environment.SetEnvironmentVariable("PATH", string.Join(";", cleanPath), EnvironmentVariableTarget.Machine);
            Environment.SetEnvironmentVariable("PATH_ORIGINAL", null, EnvironmentVariableTarget.Machine);

            return ActionResult.Success;
        }
    }
}
