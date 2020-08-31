using Microsoft.Deployment.WindowsInstaller;
using System;
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
    }
}
