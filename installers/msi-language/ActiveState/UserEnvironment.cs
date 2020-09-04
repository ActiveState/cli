using Microsoft.Deployment.WindowsInstaller;
using Microsoft.Win32;
using System;
using System.Collections.Generic;
using System.Linq;

namespace ActiveState
{
    internal static class UserEnvironment
    {
        internal static List<string> GetInstalledApps(Session session)
        {
            var res = new List<string>();
            string uninstallKey = @"SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall";
            try
            {
                using (RegistryKey rk = Registry.LocalMachine.OpenSubKey(uninstallKey))
                {
                    foreach (string skName in rk.GetSubKeyNames())
                    {
                        using (RegistryKey sk = rk.OpenSubKey(skName))
                        {
                            try
                            {
                                var val = sk.GetValue("DisplayName");
                                res.Add(val.ToString());
                            }
                            catch (Exception ex)
                            {
                                session.Log("Error retrieving installed app {0}", ex.ToString());
                            }
                        }
                    }
                }
            } catch (Exception ex)
            {
                session.Log("Error getting installed applications: {0}", ex.ToString());
            }

            return res.Distinct().ToList();
        }

        internal static List<string> GetRunningProcesses(Session session)
        {
            var res = new List<string>();
            try
            {
                var processList = System.Diagnostics.Process.GetProcesses();
                foreach (var p in processList)
                {
                    res.Add(p.ProcessName);
                }
            } catch (Exception ex)
            {
                session.Log("Error getting other running processes: {0}", ex.ToString());
            }
            return res.Distinct().ToList();
        }
    }
}
