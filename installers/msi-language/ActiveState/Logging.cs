using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Collections.Generic;
using System.IO;

namespace ActiveState
{
    public class Logging
    {
        public static string GetLog(Session session)
	    {
            string _logFileName = "";

            // Check if we are running in a deferred custom action
            if (session.GetMode(InstallRunMode.Scheduled) && session.CustomActionData.ContainsKey("MsiLogFileLocation"))
            {
                _logFileName = session.CustomActionData["MsiLogFileLocation"];
            }
            else if (!session.GetMode(InstallRunMode.Scheduled))
            {
                _logFileName = session["MsiLogFileLocation"];
            }

            if (!File.Exists(_logFileName))
	        {
                return "Log file did not exist";
	        }
            try
            {
                // open log file with shared access
                using (var fs = File.Open(_logFileName, FileMode.Open, FileAccess.Read, FileShare.ReadWrite))
		        {
                    using (var sr = new StreamReader(fs))
                    {
                        return sr.ReadToEnd();
                    }
		        }
            } 
            catch (Exception err)
	        {
                return String.Format("Error reading from log file: {0}", err.ToString());
	        }
	    }

        public static string GetProperties(Session session)
        {
            if (session.GetMode(InstallRunMode.Scheduled))
            {
                return session.CustomActionData.ToString();
            }
            // Property data is not available for immediate custom actions
            return "";
        }

        public static bool PrivacyAgreementAccepted(Session session)
        {
            string accepted;
            if (session.GetMode(InstallRunMode.Scheduled))
            {
                if (!session.CustomActionData.TryGetValue("PRIVACY_ACCEPTED", out accepted))
                {
                    accepted = "0";
                }
            } else
            {
                accepted = session["PRIVACY_ACCEPTED"];
            }
            session.Log("Privacy consent seen? {0}", accepted == "1");
            return accepted == "1";
        }

        public static Dictionary<string, object> GetUserEnvironment(Session session)
        {
            session.Log("Gather information on user environment");
            var res = new Dictionary<string, object>();
            if (!PrivacyAgreementAccepted(session))
            {
                return null;
            }

            res.Add("installed_apps", UserEnvironment.GetInstalledApps(session));
            res.Add("running_programs", UserEnvironment.GetRunningProcesses(session));
            return res;
        }
    }
}
