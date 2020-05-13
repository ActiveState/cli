using System;
using System.IO;
using Microsoft.Deployment.WindowsInstaller;

namespace GetProjectName
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult GetProjectName(Session session)
        {
            session.Log("Attempting to get Project namespace from filename");

            string filename = Path.GetFileNameWithoutExtension(session["OriginalDatabase"]);
            byte[] data = Convert.FromBase64String(filename);
            string projectName = System.Text.Encoding.Default.GetString(data);

            session.Log(string.Format("MSI filename: {0}", filename));
            session.Log(string.Format("Setting project name to {0}", projectName));
            session["PROJECT_NAME"] = projectName;
            return ActionResult.Success;
        }
    }
}
