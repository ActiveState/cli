using System;
using Microsoft.Win32;
using Microsoft.Deployment.WindowsInstaller;


namespace ActiveState
{
    public class Network
    {
        private const string networkErrorKey = "NetworkError";
        private const string networkErrorMessageKey = "NetworkErrorMessage";

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
                Registry.SetValue(registryKey, networkErrorKey, "false", registryEntryDataType);
                Registry.SetValue(registryKey, networkErrorMessageKey, "", registryEntryDataType);
            }
            catch (Exception e)
            {
                string msg = string.Format("Could not delete network error registry keys. Exception: {0}", e.ToString());
                session.Log(msg);
                RollbarReport.Error(msg, session);
            }
        }

        /// <summary>
        /// SetErrorDetails writes the network error details to the user's registry.
        /// This function must be run from a deferred custom action
        /// </summary>
        public static void SetErrorDetails(Session session, string msg)
        {
            SetErrorDetails(
                session,
                string.Format("HKEY_USERS\\{0}\\SOFTWARE\\ActiveState\\{1}", session.CustomActionData["USERSID"], session.CustomActionData["PRODUCT_NAME"]),
                msg
            );
        }

        /// <summary>
        /// SetErrorDetails writes the network error details to the user's registry
        /// </summary>
        private static void SetErrorDetails(Session session, string registryKey, string msg)
        {
            RegistryValueKind registryEntryDataType = RegistryValueKind.String;
            try
            {
                Registry.SetValue(registryKey, networkErrorKey, "true", registryEntryDataType);
                Registry.SetValue(registryKey, networkErrorMessageKey, msg, registryEntryDataType);
            }
            catch (Exception registryException)
            {
                string registryExceptionMsg = string.Format("Could not set network error registry values. Exception: {0}", registryException.ToString());
                session.Log(registryExceptionMsg);
                RollbarReport.Error(registryExceptionMsg, session);
            }
        }
    }
}
