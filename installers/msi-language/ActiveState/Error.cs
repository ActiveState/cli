using System;
using Microsoft.Win32;
using Microsoft.Deployment.WindowsInstaller;

namespace ActiveState
{
    public class Error
    {
        public const string TypeRegistryKey = "Error";
        public const string MessageRegistryKey = "ErrorMessage";

        /// <summary>
        /// ResetErrorDetails clears the registry entries for network errors.
        /// This function must be run from a deferred custom action
        /// </summary>
        public static void ResetErrorDetails(Session session)
        {
            // Deferred custom actions run as administrator so we have to specifically set
            // the registry key for the user using their SID in order for the value to
            // be available in later immediate custom actions
            string registryKey = string.Format("HKEY_USERS\\{0}\\SOFTWARE\\ActiveState\\{1}", session.CustomActionData["USERSID"], session.CustomActionData["PRODUCT_NAME"]);
            RegistryValueKind registryEntryDataType = RegistryValueKind.String;
            try
            {
                Registry.SetValue(registryKey, TypeRegistryKey, "", registryEntryDataType);
                Registry.SetValue(registryKey, MessageRegistryKey, "", registryEntryDataType);
            }
            catch (Exception e)
            {
                string msg = string.Format("Could not delete network error registry keys. Exception: {0}", e.ToString());
                session.Log(msg);
                RollbarReport.Error(msg, session);
            }
        }

        /// <summary>
        /// SetDetails writes the error details to the user's registry.
        /// This function must be run from a deferred custom action
        /// </summary>
        public static void SetDetails(Session session, string errorType, string msg)
        {
            string registryKey = string.Format("HKEY_USERS\\{0}\\SOFTWARE\\ActiveState\\{1}", session.CustomActionData["USERSID"], session.CustomActionData["PRODUCT_NAME"]);
            RegistryValueKind registryEntryDataType = RegistryValueKind.String;
            try
            {
                Registry.SetValue(registryKey, TypeRegistryKey, errorType, registryEntryDataType);
                Registry.SetValue(registryKey, MessageRegistryKey, msg, registryEntryDataType);
            }
            catch (Exception registryException)
            {
                string registryExceptionMsg = string.Format("Could not set error registry values. Exception: {0}", registryException.ToString());
                session.Log(registryExceptionMsg);
                RollbarReport.Error(registryExceptionMsg, session);
            }
        }
    }

    public class PathError
    {
        private const string type = "Path";

        public static void SetDetails(Session session, string msg)
        {
            Error.SetDetails(session, type, msg);
        }

        public static string Type()
        {
            return type;
        }
    }

    public class NetworkError
    {
        private const string type = "Network";

        public static void SetDetails(Session session, string msg)
        {
            Error.SetDetails(session, "Network", msg);
        }

        public static string Type()
        {
            return type;
        }
    }
}
