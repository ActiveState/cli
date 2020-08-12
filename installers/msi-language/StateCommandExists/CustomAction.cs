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

            CheckCommand(session, "state.exe", "STATE_TOOL_INSTALLED", "STATE_TOOL_PATH");
            CheckCommand(session, "code.cmd", "CODE_INSTALLED", "CODE_PATH");
            return ActionResult.Success;
        }

        private static void CheckCommand(Session session, string command, string installedProperty, string pathProperty)
        {
            session.Log(string.Format("Checking installation of: {0}", command));
            var values = Environment.GetEnvironmentVariable("PATH");
            foreach (var path in values.Split(Path.PathSeparator))
            {
                var fullPath = Path.Combine(path, command);
                if (File.Exists(fullPath))
                {
                    session[installedProperty] = "true";
                    session[pathProperty] = fullPath;
                    return;
                }
            }
            session[installedProperty] = "false";
            return;
        }
    }
}
