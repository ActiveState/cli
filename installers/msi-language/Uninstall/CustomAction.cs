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

        public static ActionResult UninstallPreset(ActiveState.Logging log)
        {
            var presetStr = log.Session().CustomActionData["PRESET"];
            var installDir = log.Session().CustomActionData["REMEMBER"];
            var shortcutDir = log.Session().CustomActionData["REMEMBER_SHORTCUTDIR"];

            var p = ParsePreset.Parse(presetStr, log, installDir, shortcutDir);

            try
            {
                return p.Uninstall();
            } catch (Exception err)
            {
                string msg = string.Format("unknown error during preset-uninstall {0}", err);
                log.Log(msg);
                RollbarReport.Error(string.Format("unknown error during uninstall: {0}", err), log);

                // We finish the uninstallation anyways, as otherwise the MSI becomes un-installable.  And that's bad!
                return ActionResult.Success;
            }
        }

        [CustomAction]
        public static ActionResult Uninstall(Session session)
        {
            ActiveState.RollbarHelper.ConfigureRollbarSingleton(session.CustomActionData["COMMIT_ID"]);
            string installDir = session.CustomActionData["REMEMBER"];

            using (var log = new ActiveState.Logging(session, installDir))
            {
                log.Log("Begin uninstallation");

                ActionResult result;
                if (installDir != "")
                {
                    result = Remove.Dir(log, installDir);
                    if (result.Equals(ActionResult.Failure))
                    {
                        log.Log("Could not remove installation directory");

                        Record record = new Record();
                        record.FormatString = string.Format("Could not remove installation directory entry at: {0}, please ensure no files in the directory are currently being used and try again", installDir);

                        session.Message(InstallMessage.Error | (InstallMessage)MessageBoxButtons.OK, record);
                        return ActionResult.Failure;
                    }

                    result = Remove.EnvironmentEntries(log, installDir);
                    if (result.Equals(ActionResult.Failure))
                    {
                        string msg = "Could not remove environment entries";
                        log.Log(msg);
                        RollbarReport.Critical(msg, log);
                        return ActionResult.Failure;
                    }
                }
                else
                {
                    log.Log("REMEMBER variable was not set in UNINSTALL");
                }

                string shortcutDir = session.CustomActionData["REMEMBER_SHORTCUTDIR"];

                if (shortcutDir != "")
                {
                    result = Remove.Dir(log, shortcutDir);
                    if (result.Equals(ActionResult.Failure))
                    {
                        string msg = "Could not remove shortcuts directory";
                        log.Log(msg);
                        RollbarReport.Critical(msg, log);
                        return ActionResult.Failure;
                    }
                }
                else
                {
                    log.Log("REMEMBER_SHORTCUTDIR was not set in UNINSTALL");
                }

                result = UninstallPreset(log);
                if (result.Equals(ActionResult.Failure))
                {
                    string msg = "Could not uninstall language preset";
                    log.Log(msg);
                    RollbarReport.Critical(msg, log);
                    return ActionResult.Failure;
                }
                return result;
            }
        }
    }
    public class Remove
    {
        public static ActionResult Dir(ActiveState.Logging log, string dir)
        {
            log.Log(string.Format("Removing directory: {0}", dir));

            if (Directory.Exists(dir))
            {
                try
                {
                    Directory.Delete(dir, true);
                }
                catch (Exception e)
                {
                    string msg = string.Format("Could not delete install directory, got error: {0}", e.ToString());
                    log.Log(msg);
                    RollbarReport.Critical(msg, log);
                    return ActionResult.Failure;
                }
            }

            return ActionResult.Success;
        }

        public static ActionResult EnvironmentEntries(ActiveState.Logging log, string dir)
        {
            log.Log("Begin removing environment entries");
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
