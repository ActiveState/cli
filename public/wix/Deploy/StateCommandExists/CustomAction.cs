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

            var values = Environment.GetEnvironmentVariable("PATH");
            foreach (var path in values.Split(Path.PathSeparator))
            {
                var fullPath = Path.Combine(path, "state.exe");
                if (File.Exists(fullPath))
                    return ActionResult.Success;
            }
            session.Message(InstallMessage.Error, new Record { FormatString = "State Tool installation does not exist on system, please install the State Tool and try again." });
            return ActionResult.Failure;
        }
    }
}
