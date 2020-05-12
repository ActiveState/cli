using System;
using System.IO;
using System.Net;
using Microsoft.Deployment.WindowsInstaller;
using Newtonsoft.Json;

namespace GetProjectName
{
    public class Project
    {
        public string Name { get; set; }
        public string OrganizationID { get; set; }

    }

    public class Organization
    {
        public string URLName { get; set; }
    }
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult GetProjectName(Session session)
        {
            session.Log("Attempting to get Project namespace from filename");

            if (!LoggedIn(session))
            {
                session.Message(InstallMessage.Error, new Record { FormatString = "You must be logged in with the State Tool in order to install a language runtime. Please run `state auth` and try again." });
                return ActionResult.Failure;
            }

            var msiFullName = session["OriginalDatabase"];
            session.Log("msiFullName: " + msiFullName);

            // TODO: Use msiFullName
            var projectID = Path.GetFileNameWithoutExtension(msiFullName);

            string token = GetToken(session);
            if (token == "") {
                session.Log("Could not get JWT from State Tool");
                return ActionResult.Failure;
            }

            Project project = GetProject(session, projectID, token);
            Organization organization = GetOrganization(session, project.OrganizationID, token);

            session.Log(string.Format("Setting project name to {0}/{1}", organization.URLName, project.Name));
            session["PROJECT_NAME"] = string.Format("{0}/{1}", organization.URLName, project.Name);
            return ActionResult.Success;
        }

        public static string GetToken(Session session)
        {
            string token = "";
            try
            {
                System.Diagnostics.ProcessStartInfo procStartInfo =
                    new System.Diagnostics.ProcessStartInfo("cmd", "/c " + "state export jwt");

                // The following commands are needed to redirect the standard output.
                // This means that it will be redirected to the Process.StandardOutput StreamReader.
                procStartInfo.RedirectStandardOutput = true;
                procStartInfo.UseShellExecute = false;
                // Do not create the black window.
                procStartInfo.CreateNoWindow = true;

                System.Diagnostics.Process proc = new System.Diagnostics.Process();
                proc.StartInfo = procStartInfo;
                proc.Start();
                token = proc.StandardOutput.ReadToEnd();
            }
            catch (Exception objException)
            {
                session.Log(string.Format("Caught exception: {0}", objException));
            }

            return token;
        }

        public static bool LoggedIn(Session session)
        {
            string output = "";
            try
            {
                System.Diagnostics.ProcessStartInfo procStartInfo =
                    new System.Diagnostics.ProcessStartInfo("cmd", "/c " + "state auth");

                // The following commands are needed to redirect the standard output.
                // This means that it will be redirected to the Process.StandardOutput StreamReader.
                procStartInfo.RedirectStandardOutput = true;
                procStartInfo.UseShellExecute = false;
                // Do not create the black window.
                procStartInfo.CreateNoWindow = true;

                System.Diagnostics.Process proc = new System.Diagnostics.Process();
                proc.StartInfo = procStartInfo;
                proc.Start();
                output = proc.StandardOutput.ReadToEnd();
            }
            catch (Exception objException)
            {
                session.Log(string.Format("Caught exception: {0}", objException));
            }

            return output.Contains("You are logged in as");
        }

        public static Project GetProject(Session session, string projectID, string token)
        {
            // TODO: Use projectID
            string projectsURL = string.Format("https://platform.activestate.com/api/v1/projects/{0}", "d5414d38-6d12-4d2a-afd5-571e8cdbeaa5");
            HttpWebRequest projectReq = (HttpWebRequest)WebRequest.Create(projectsURL);

            string authorizationHeader = string.Format("Authorization: {0}", token);
            projectReq.Headers.Add(authorizationHeader);
            projectReq.ContentType = "application/json";

            WebResponse projectResponse = projectReq.GetResponse();
            session.Log(((HttpWebResponse)projectResponse).StatusDescription);

            string projectName;
            string organizationID;
            Project project;
            using (Stream dataStream = projectResponse.GetResponseStream())
            {
                StreamReader reader = new StreamReader(dataStream);
            
                string responseFromServer = reader.ReadToEnd();

                project = JsonConvert.DeserializeObject<Project>(responseFromServer);

                session.Log(string.Format("Project name: {0}", project.Name));
                session.Log(string.Format("Organization ID: {0}", project.OrganizationID));
                projectName = project.Name;
                organizationID = project.OrganizationID;
            }

            projectResponse.Close();

            return project;
        }

        public static Organization GetOrganization(Session session, string organizationID, string token)
        {
            var orgsURL = string.Format("https://platform.activestate.com/api/v1/organizations/{0}?identifierType=organizationID", organizationID);
            HttpWebRequest orgReq = (HttpWebRequest)WebRequest.Create(orgsURL);

            string authorizationHeader = string.Format("Authorization: {0}", token);
            orgReq.Headers.Add(authorizationHeader);
            orgReq.ContentType = "application/json";

            WebResponse orgResponse = orgReq.GetResponse();
            session.Log(((HttpWebResponse)orgResponse).StatusDescription);

            Organization organization;
            using (Stream dataStream = orgResponse.GetResponseStream())
            {
                StreamReader reader = new StreamReader(dataStream);

                string responseFromServer = reader.ReadToEnd();

                organization = JsonConvert.DeserializeObject<Organization>(responseFromServer);

                session.Log(string.Format("Organization name: {0}", organization.URLName));
            }

            orgResponse.Close();

            return organization;
        }
    }
}
