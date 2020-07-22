using System;
using System.IO;
using Microsoft.Deployment.WindowsInstaller;

namespace StateCommandExists
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult StateCommandExists(Session session)
        {
            session.Log("Checking State Tool installation");

            ActiveState.RollbarHelper.ConfigureRollbarSingleton();

            var values = Environment.GetEnvironmentVariable("PATH");
            foreach (var path in values.Split(Path.PathSeparator))
            {
                var fullPath = Path.Combine(path, "state.exe");
                if (File.Exists(fullPath))
                {
                    session["STATE_TOOL_INSTALLED"] = "true";
                    session["STATE_TOOL_PATH"] = fullPath;
                    return ActionResult.Success;
                }
            }
            session["STATE_TOOL_INSTALLED"] = "false";
            return ActionResult.Success;
        }
    }
}
