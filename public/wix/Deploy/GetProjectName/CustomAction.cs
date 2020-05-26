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
            session.Log("Attempting to get Project namespace from MSI filename");

            string filename = Path.GetFileNameWithoutExtension(session["OriginalDatabase"]);
            session.Log(string.Format("MSI filename: {0}", filename));

            Record installErrorRecord = new Record
            {
                FormatString = string.Format("Invalid filename found: {0}. Filename must be of the format <LanguageName><Version>-<Base64 String>", filename)
            };

            string[] parts = filename.Split('-');
            if (parts.Count() != 2)
            {
                session.Message(InstallMessage.Error, installErrorRecord);
                return ActionResult.Failure;
            }

            byte[] data;
            try
            {
                data = Convert.FromBase64String(parts[1]);
            }
            catch (Exception e)
            {
                session.Log(string.Format("Could not convert base64 filename, got error: {0}", e.ToString()));
                session.Message(InstallMessage.Error, installErrorRecord);
                return ActionResult.Failure;
            }

            string projectNamespace;
            try
            {
                projectNamespace = System.Text.Encoding.Default.GetString(data);
            }
            catch (Exception e)
            {
                session.Log(string.Format("Could not decode filename to project namespace, got error: {0}", e.ToString()));
                session.Message(InstallMessage.Error, installErrorRecord);
                return ActionResult.Failure;
            }
            

            // Set PROJECT_NAME to be used in other custom actions
            session.Log(string.Format("Setting PROJECT_NAME to: {0}", projectNamespace));
            session["PROJECT_NAME"] = projectNamespace;

            // Update INSTALLDIR to dynamically set the installation directory to the project name
            return SetInstallDir(session, projectNamespace);
        }

        private static ActionResult SetInstallDir(Session session, string projectNamespace)
        {
            string originalInstallDir = session["INSTALLDIR"];
            string[] elements = Path.GetDirectoryName(originalInstallDir).Split('\\');
            if (elements.Count() <= 0)
            {
                session.Message(InstallMessage.Error, new Record { FormatString = string.Format("Invalid install directory: {0}.", originalInstallDir) });
                return ActionResult.Failure;
            }

            string projectName = projectNamespace.Substring(projectNamespace.IndexOf('/') + 1);
            elements[elements.Count() - 1] = projectName;
            string updatedInstallDir = string.Join("\\", elements);

            session.Log(string.Format("Setting INSTALLDIR to: {0}", updatedInstallDir));
            session["INSTALLDIR"] = updatedInstallDir;
            return ActionResult.Success;
        }
    }
}
