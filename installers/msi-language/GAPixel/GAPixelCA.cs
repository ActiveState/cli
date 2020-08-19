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

        public static string GetUniqueId(Session session)
        {
            try
            {
                var oMClass = new ManagementClass("Win32_NetworkAdapterConfiguration");
                var colMObj = oMClass.GetInstances();
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
                        session.Log(String.Format("Found MAC={0}", macAddress));
                        return StringToGuid(macAddress.ToString()).ToString();
                    } catch(NullReferenceException)
                    {
                        continue;
                    }
                }
            }
            catch (Exception err)
            {
                session.Log(String.Format("Error getting unique ID {0}", err));
            }
            // fallback method
            return Guid.NewGuid().ToString();
        }
    }



    // modified from https://docs.microsoft.com/en-us/dotnet/framework/migration-guide/how-to-determine-which-versions-are-installed#query-the-registry-using-code
    // and from https://docs.microsoft.com/en-us/dotnet/framework/migration-guide/how-to-determine-which-versions-are-installed#query-the-registry-using-code-older-framework-versions
    public class GetDotNetVersion
    {
        private struct DotNetVersion
        {
            public DotNetVersion(string SP, string version)
            {
                this.ServicePack = SP;
                this.Version = new Version(version);
            }

            public override string ToString()
            {
                if (String.IsNullOrEmpty(this.ServicePack))
                {
                    return this.Version.ToString();
                }
                return String.Format("{0}-SP{1}", this.Version.ToString(), this.ServicePack);
            }

            public string ServicePack;
            public Version Version;
        }

        private static string GetLatestVersionFromRegistryLessThanV45()
        {
            var versions = new List<DotNetVersion>();
            // Opens the registry key for the .NET Framework entry.
            using (RegistryKey ndpKey =
                    Registry.LocalMachine.OpenSubKey(@"SOFTWARE\Microsoft\NET Framework Setup\NDP\"))
            {
                foreach (var versionKeyName in ndpKey.GetSubKeyNames())
                {
                    // Skip .NET Framework 4.5 version information.
                    if (versionKeyName == "v4")
                    {
                        continue;
                    }

                    if (versionKeyName.StartsWith("v"))
                    {

                        RegistryKey versionKey = ndpKey.OpenSubKey(versionKeyName);
                        // Get the .NET Framework version value.
                        var name = (string)versionKey.GetValue("Version", "");
                        // Get the service pack (SP) number.
                        var sp = versionKey.GetValue("SP", "").ToString();

                        // Get the installation flag, or an empty string if there is none.
                        var install = versionKey.GetValue("Install", "").ToString();
                        if (!string.IsNullOrEmpty(install))
                        {
                            if (!(string.IsNullOrEmpty(sp)) && install == "1")
                            {
                                versions.Add(new DotNetVersion(sp, name));
                            }
                        }
                        if (!string.IsNullOrEmpty(name))
                        {
                            continue;
                        }
                        foreach (var subKeyName in versionKey.GetSubKeyNames())
                        {
                            RegistryKey subKey = versionKey.OpenSubKey(subKeyName);
                            name = (string)subKey.GetValue("Version", "");
                            if (!string.IsNullOrEmpty(name))
                                sp = subKey.GetValue("SP", "").ToString();

                            install = subKey.GetValue("Install", "").ToString();
                            if (!string.IsNullOrEmpty(install))
                            {
                                if (!(string.IsNullOrEmpty(sp)) && install == "1")
                                {
                                    versions.Add(new DotNetVersion(sp, name));
                                }
                                else if (install == "1")
                                {
                                    versions.Add(new DotNetVersion("", name));
                                }
                            }
                        }
                    }
                }
            }

            var maxVersion = new DotNetVersion("", "0.0.0");
            foreach (var v in versions)
            {
                if (v.Version > maxVersion.Version)
                {
                    maxVersion = v;
                }
                if (v.Version == maxVersion.Version && v.ServicePack.CompareTo(maxVersion.ServicePack) > 0)
                {
                    maxVersion = v;
                }
            }
            return maxVersion.ToString();
        }

        public static string GetLatestVersion()
        {
            const string subkey = @"SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full\";

            using (var ndpKey = Registry.LocalMachine.OpenSubKey(subkey))
            {
                if (ndpKey != null && ndpKey.GetValue("Release") != null)
                {
                    return CheckFor45PlusVersion((int)ndpKey.GetValue("Release"));
                }
                else
                {
                    return GetLatestVersionFromRegistryLessThanV45();
                }
            }

            // Checking the version using >= enables forward compatibility.
            string CheckFor45PlusVersion(int releaseKey)
            {
                if (releaseKey >= 528040)
                    return "4.8+";
                if (releaseKey >= 461808)
                    return "4.7.2";
                if (releaseKey >= 461308)
                    return "4.7.1";
                if (releaseKey >= 460798)
                    return "4.7";
                if (releaseKey >= 394802)
                    return "4.6.2";
                if (releaseKey >= 394254)
                    return "4.6.1";
                if (releaseKey >= 393295)
                    return "4.6";
                if (releaseKey >= 379893)
                    return "4.5.2";
                if (releaseKey >= 378675)
                    return "4.5.1";
                if (releaseKey >= 378389)
                    return "4.5";
                // This code should never execute. A non-null release key should mean
                // that 4.5 or later is installed.
                return "unknown";
            }
        }
    }
    // This example displays output like the following:
    //      
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
            var dnv = GetDotNetVersion.GetLatestVersion();
            var wv = System.Environment.OSVersion.VersionString;
            string cid = GetInfo.GetUniqueId(session);

            session.Log(String.Format("Send Pixel to GA windows version={0}, latest dotNet version={1}", wv, dnv)); ;

            /* I tried running the following in a thread, but unfortunately that thread got cancelled once the function returned.
             * So, the dialog will "hang" in the beginning until the analytics tracking event has been send or timed out.
             */
            var client = new WebClient();
            // set the timeout to 5 seconds
            client.Timeout = 5 * 1000;
            session.Log(String.Format("In Thread: send pixel to GA windows version={0}, latest dotNet version={1} cid={2}", wv, dnv, cid)); ;

            // Call asynchronous network methods in a try/catch block to handle exceptions.
            try
            {
                var s = client.DownloadString(String.Format(@"http://localhost:8000/collect?v=1&t=event&tid=UA-118120158-1&cid={0}&ec=old_dotnet&ea={1}&el={2}",
                        cid, Uri.EscapeUriString(wv), Uri.EscapeUriString(dnv)));
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
