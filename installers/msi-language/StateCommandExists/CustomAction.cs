using System;
using System.IO;
using Microsoft.Deployment.WindowsInstaller;

namespace StateCommandExists
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult CheckCommands(Session session)
        { 
            ActiveState.RollbarHelper.ConfigureRollbarSingleton(session["COMMIT_ID"]);
            string installDir = session["INSTALLDIR"];
            using (var log = new ActiveState.Logging(session, installDir))
            {

                CheckCommand(log, "state.exe", "STATE_TOOL_INSTALLED", "STATE_TOOL_PATH");
                CheckCommand(log, "code.cmd", "CODE_INSTALLED", "CODE_PATH");
                return ActionResult.Success;
            }
        }

        private static void CheckCommand(ActiveState.Logging log, string command, string installedProperty, string pathProperty)
        {
            log.Log(string.Format("Checking installation of: {0}", command));
            var values = Environment.GetEnvironmentVariable("PATH");
            foreach (var path in values.Split(Path.PathSeparator))
            {
                var fullPath = Path.Combine(path, command);
                if (File.Exists(fullPath))
                {
                    log.Session()[installedProperty] = "true";
                    log.Session()[pathProperty] = fullPath;
                    return;
                }
            }
            log.Session()[installedProperty] = "false";
            log.Log("Did not find {0}", command);
            return;
        }
    }
}
