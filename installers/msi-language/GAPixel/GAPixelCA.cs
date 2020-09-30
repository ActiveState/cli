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

        private static Guid StringToGuid(string input)
        {
            using (System.Security.Cryptography.MD5 md5 = System.Security.Cryptography.MD5.Create())
            {
                byte[] inputBytes = System.Text.Encoding.ASCII.GetBytes(input);
                byte[] hashBytes = md5.ComputeHash(inputBytes);

                return new Guid(hashBytes);
            }
        }

        public static string DecodeProductKeyWin8AndUp(byte[] digitalProductId)
        {
            var key = String.Empty;
            const int keyOffset = 52;
            var isWin8 = (byte)((digitalProductId[66] / 6) & 1);
            digitalProductId[66] = (byte)((digitalProductId[66] & 0xf7) | (isWin8 & 2) * 4);

            // Possible alpha-numeric characters in product key.
            const string digits = "BCDFGHJKMPQRTVWXY2346789";
            int last = 0;
            for (var i = 24; i >= 0; i--)
            {
                var current = 0;
                for (var j = 14; j >= 0; j--)
                {
                    current = current * 256;
                    current = digitalProductId[j + keyOffset] + current;
                    digitalProductId[j + keyOffset] = (byte)(current / 24);
                    current = current % 24;
                    last = current;
                }
                key = digits[current] + key;
            }
            var keypart1 = key.Substring(1, last);
            const string insert = "N";
            key = key.Substring(1).Replace(keypart1, keypart1 + insert);
            if (last == 0)
                key = insert + key;
            for (var i = 5; i < key.Length; i += 6)
            {
                key = key.Insert(i, "-");
            }
            return key;
        }

        public static string GetWindowsProductKey(Session session=null)
        {
            try
            {
                const string keyPath = @"Software\Microsoft\Windows NT\CurrentVersion";
                var digitalProductId = (byte[])Registry.LocalMachine.OpenSubKey(keyPath).GetValue("DigitalProductId");

                var isWin8OrUp =
                    (Environment.OSVersion.Version.Major == 6 && System.Environment.OSVersion.Version.Minor >= 2)
                    ||
                    (Environment.OSVersion.Version.Major > 6);

                var productKey = isWin8OrUp ? DecodeProductKeyWin8AndUp(digitalProductId) : "windows < 8";
                session.Log("Successfully decoded windows product key {0}", productKey);
                return productKey;
            }
            catch (Exception err)
            {
                if (session != null)
                {
                    session.Log("Error getting windows product key: {0}", err);
                }
                return "error";
            }
        }

        public static string GetUniqueId(Session session=null)
        {
            try
            {
                var oMClass = new ManagementClass("Win32_NetworkAdapterConfiguration");
                var colMObj = oMClass.GetInstances();
                var productKey = GetWindowsProductKey(session);
                foreach (var objMO in colMObj)
                {
                    try
                    {
                        var macAddress = objMO["MacAddress"].ToString();
                        if (String.IsNullOrEmpty(macAddress))
                        {
                            continue;
                        }
                        // return on first found MAC address
                        return StringToGuid(macAddress.ToString() + productKey).ToString();
                    }
                    catch (NullReferenceException)
                    {
                        continue;
                    }
                }
            }
            catch (Exception err)
            {
                if (session != null)
                {
                    session.Log(String.Format("Error getting unique ID {0}", err));
                }
            }
            // fallback GUID
            return "11111111--1111-1111-1111-111111111111";
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
            string cid = GetInfo.GetUniqueId(session);

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
