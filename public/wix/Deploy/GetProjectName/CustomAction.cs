using System;
using System.IO;
using System.Linq;
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
            session.Log(string.Format("MSI filename: {0}", filename));

            string[] parts = filename.Split('-');
            if (parts.Count() != 2)
            {
                session.Message(InstallMessage.Error, new Record{ FormatString = string.Format("Invalid filename found: {0}. Filename must be of the format <LanguageName><Version>-<Base64 String>", filename)});
                return ActionResult.Failure;
            }

            byte[] data = Convert.FromBase64String(parts[1]);
            string projectNamespace = System.Text.Encoding.Default.GetString(data);

            // Set PROJECT_NAME to be used in other custom actions
            session.Log(string.Format("Setting PROJECT_NAME to: {0}", projectNamespace));
            session["PROJECT_NAME"] = projectNamespace;

            // Update INSTALLDIR to dynamically set the installation directory to the project name
            string originalInstallDir = session["INSTALLDIR"];
            string[] elements = Path.GetDirectoryName(originalInstallDir).Split('\\');

            string projectName = projectNamespace.Substring(projectNamespace.IndexOf('/') + 1);
            elements[elements.Count() - 1] = projectName;
            string updatedInstallDir = string.Join("\\", elements);

            session.Log(string.Format("Setting INSTALLDIR to: {0}", updatedInstallDir));
            session["INSTALLDIR"] = updatedInstallDir;
            return ActionResult.Success;
        }
    }
}
