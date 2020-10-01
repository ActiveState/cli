using System;
using System.Collections.Generic;
using Microsoft.Deployment.WindowsInstaller;
using System.Management;
using System.Net;
using Microsoft.Win32;


namespace GAPixel
{
    public class GetInfo
    {

        public static string GetOrCreateNewCid(Session session)
        {
            try
            {
                var baseKey = Registry.CurrentUser;
                if (session.GetMode(InstallRunMode.Scheduled) && session.CustomActionData.ContainsKey("USERSID"))
                {
                    baseKey = Registry.Users.OpenSubKey(session.CustomActionData["USERSID"], true);
                }
                var keyPath = @"Software\ActiveState";
                var key = baseKey.CreateSubKey(keyPath);
                var cidObj = key.GetValue("CID");
                string cid;
                if (cidObj != null)
                {
                    cid = cidObj.ToString();
                }
                else {
                    cid = Guid.NewGuid().ToString();
                    key.SetValue("CID", cid, RegistryValueKind.String);
                }
                return cid;
            }
            catch (Exception err)
            {
                if (session != null)
                {
                    session.Log("Error creating or getting CID: {0}", err);
                }
                // fallback GUID
                return "11111111--1111-1111-1111-111111111111";
            }
        }
    }

    public class CustomActions
    {
        // Modify WebClient so we can set a timeout
        private class WebClient : System.Net.WebClient
        {
            public int Timeout { get; set; }

            protected override WebRequest GetWebRequest(Uri uri)
            {
                WebRequest lWebRequest = base.GetWebRequest(uri);
                lWebRequest.Timeout = Timeout;
                ((HttpWebRequest)lWebRequest).ReadWriteTimeout = Timeout;
                return lWebRequest;
            }
        }

        [CustomAction]
        public static ActionResult SendPixel(Session session)
        {
            var wv = System.Environment.OSVersion.VersionString;
            string cid = GetInfo.GetOrCreateNewCid(session);

            session.Log(String.Format("Send Pixel to GA windows version={0}", wv)); ;

            /* I tried running the following in a thread, but unfortunately that thread got cancelled once the function returned.
             * So, the dialog will "hang" in the beginning until the analytics tracking event has been send or timed out.
             */
            var client = new WebClient();
            // set a low timeout of 10 seconds
            client.Timeout = 10 * 1000;

            // Call asynchronous network methods in a try/catch block to handle exceptions.
            try
            {
                var s = client.DownloadString(String.Format(
                        @"https://ssl.google-analytics.com/collect?v=1&t=event&tid=UA-118120158-2&cid={0}&ec=old_dotnet&ea={1}",
                        cid, Uri.EscapeUriString(wv)));
                session.Log(String.Format("GA resonded with {0}", s));
            }
            catch (WebException e)
            {
                session.Log("Error sending tracking pixel: {0}", e.Message);
            }

            return ActionResult.Success;
        }
    }
}
