using System;
using System.IO;
using System.Net;
using Microsoft.Deployment.WindowsInstaller;
using System.Diagnostics;

namespace InstallStateTool
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult InstallStateTool(Session session)
        {
            session.Log("Installing State Tool if necessary");
            if (session.CustomActionData["STATE_TOOL_INSTALLED"] == "true")
            {
                session.Log("State Tool is installed, no installation required");
                return ActionResult.Success;
            }

            string tempDir = Path.GetTempPath();
            string scriptPath = Path.Combine(tempDir, "install.ps1");
            string installPath = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "ActiveState", "bin");

            Status.ProgressBar.StatusMessage(session, "Installing State Tool...");
            ServicePointManager.SecurityProtocol |= SecurityProtocolType.Tls11 | SecurityProtocolType.Tls12;

            try
            {
                WebClient client = new WebClient();
                client.DownloadFile("https://platform.activestate.com/dl/cli/install.ps1", scriptPath);
            } catch (WebException e)
            {
                session.Log(string.Format("Encoutered exception downloading file: {0}", e.ToString()));
                return ActionResult.Failure;
            }


            string installCmd = string.Format("powershell \"{0} -n -t {1}\"", scriptPath, installPath);
            session.Log(string.Format("Running install command: {0}", installCmd));
            ActionResult result = RunCommand(session, installCmd);
            if (result.Equals(ActionResult.UserExit))
            {
                result = Uninstall.Remove.InstallDir(session, installPath);
                if (result.Equals(ActionResult.Failure))
                {
                    session.Log("Could not remove installation directory");
                    return ActionResult.Failure;
                }

                result = Uninstall.Remove.EnvironmentEntries(session, installPath);
                if (result.Equals(ActionResult.Failure))
                {
                    session.Log("Could not remove environment entries");
                    return ActionResult.Failure;
                }
                return ActionResult.UserExit;
            }

            session["STATE_TOOL_PATH"] = Path.Combine(installPath, "state.exe");
            return result;
        }

        private static ActionResult RunCommand(Session session, string cmd)
        {
            try
            {
                ProcessStartInfo procStartInfo = new ProcessStartInfo("cmd", "/c " + cmd);

                // The following commands are needed to redirect the standard output.
                // This means that it will be redirected to the Process.StandardOutput StreamReader.
                procStartInfo.RedirectStandardOutput = true;
                procStartInfo.RedirectStandardError = true;
                procStartInfo.UseShellExecute = false;
                // Do not create the black window.
                procStartInfo.CreateNoWindow = true;

                Process proc = new Process();
                proc.StartInfo = procStartInfo;
                proc.Start();

                while (!proc.HasExited)
                {
                    try
                    {
                        Status.ProgressBar.Increment(session, 0);
                        System.Threading.Thread.Sleep(200);
                    }
                    catch (InstallCanceledException)
                    {
                        session.Log("Caught install cancelled exception");
                        ActiveState.Process.KillProcessAndChildren(proc.Id);
                        return ActionResult.UserExit;
                    }
                }
                proc.WaitForExit();
                session.Log(string.Format("Standard output: {0}", proc.StandardOutput.ReadToEnd()));
                session.Log(string.Format("Standard error: {0}", proc.StandardError.ReadToEnd()));
            }
            catch (Exception objException)
            {
                session.Log(string.Format("Caught exception: {0}", objException));
                return ActionResult.Failure;
            }
            return ActionResult.Success;
        }

    }
}
