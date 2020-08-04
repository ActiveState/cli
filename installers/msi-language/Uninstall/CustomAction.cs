using System;
using System.Collections.Generic;
using System.IO;
using System.Windows.Forms;
using ActiveState;
using Microsoft.Deployment.WindowsInstaller;
using Preset;

namespace Uninstall
{
    public class CustomActions
    {

        public static ActionResult UninstallPreset(Session session)
        {
            var presetStr = session.CustomActionData["PRESET"];
            var installDir = session.CustomActionData["REMEMBER"];
            var shortcutDir = session.CustomActionData["REMEMBER_SHORTCUTDIR"];

            var p = ParsePreset.Parse(presetStr, session, installDir, shortcutDir);

            try
            {
                return p.Uninstall();
            } catch (Exception err)
            {
                session.Log(string.Format("unknown error during preset-uninstall {0}", err));
                RollbarHelper.Report(string.Format("unknown error during uninstall: {0}", err));

                // We finish the uninstallation anyways, as otherwise the MSI becomes un-installable.  And that's bad!
                return ActionResult.Success;
            }
        }

        [CustomAction]
        public static ActionResult Uninstall(Session session)
        {
            ActiveState.RollbarHelper.ConfigureRollbarSingleton();

            session.Log("Begin uninstallation");

            ActionResult result;
            string installDir = session.CustomActionData["REMEMBER"];
            if (installDir != "")
            {
                result = Remove.Dir(session, installDir);
                if (result.Equals(ActionResult.Failure))
                {
                    session.Log("Could not remove installation directory");

                    Record record = new Record();
                    record.FormatString = string.Format("Could not remove installation directory entry at: {0}, please ensure no files in the directory are currently being used and try again", installDir);

                    session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                    return ActionResult.Failure;
                }

                result = Remove.EnvironmentEntries(session, installDir);
                if (result.Equals(ActionResult.Failure))
                {
                    session.Log("Could not remove environment entries");
                    ActiveState.RollbarHelper.Report("Could not remove environment entries");
                    return ActionResult.Failure;
                }
            } else
            {
                session.Log("REMEMBER variable was not set in UNINSTALL");
            }

            string shortcutDir = session.CustomActionData["REMEMBER_SHORTCUTDIR"];

            if (shortcutDir != "")
            {
                result = Remove.Dir(session, shortcutDir);
                if (result.Equals(ActionResult.Failure))
                {
                    session.Log("Could not remove shortcuts directory");
                    ActiveState.RollbarHelper.Report("Could not remove shortcuts directory");
                    return ActionResult.Failure;
                }
            } else
            {
                session.Log("REMEMBER_SHORTCUTDIR was not set in UNINSTALL");
            }

            result = UninstallPreset(session);
            if (result.Equals(ActionResult.Failure))
            {
                session.Log("Could not uninstall language preset");
                ActiveState.RollbarHelper.Report("Could not uninstall language preset");
                return ActionResult.Failure;
            }
            return result;
        }
    }
    public class Remove
    {
        public static ActionResult Dir(Session session, string dir)
        {
            session.Log(string.Format("Removing directory: {0}", dir));

            if (Directory.Exists(dir))
            {
                try
                {
                    Directory.Delete(dir, true);
                }
                catch (Exception e)
                {
                    session.Log(string.Format("Could not delete install directory, got error: {0}", e.ToString()));
                    ActiveState.RollbarHelper.Report(string.Format("Could not delete install directory, got error: {0}", e.ToString()));
                    return ActionResult.Failure;
                }
            }

            return ActionResult.Success;
        }

        public static ActionResult EnvironmentEntries(Session session, string dir)
        {
            session.Log("Begin removing environment entries");
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

            return ActionResult.Success;
        }
    }
}
