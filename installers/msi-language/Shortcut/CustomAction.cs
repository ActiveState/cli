using System;
using System.IO;
using IWshRuntimeLibrary;
using Microsoft.Deployment.WindowsInstaller;

namespace Shortcut
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult InstallShortcuts(Session session)
        {
            session.Log("Begin InstallShortcuts");

            ActiveState.RollbarHelper.ConfigureRollbarSingleton();

            string shortcutData = session.CustomActionData["SHORTCUTS"];
            string appStartMenuPath = session.CustomActionData["APP_START_MENU_PATH"];
            if (shortcutData.ToLower() == "none")
            {
                session.Log("Recieved none, not building any shortcuts");
                return ActionResult.Success;
            }

            string[] shortcuts = shortcutData.Split(',');
            foreach (string shortcut in shortcuts)
            {
                var s = shortcut.ToLower();
                switch (s)
                {
                    case "perlcritic":
                        {
                            ActionResult result = PerlCriticShortcut(session, appStartMenuPath);
                            if (result.Equals(ActionResult.Failure))
                            {
                                session.Log("Could not create Perl Critic shortcut");
                                // Do not fail if we cannot create shortcut
                                return ActionResult.Success;
                            }
                            break;
                        }
                    case "cmdprompt":
                        {
                            ActionResult result = CmdPromptShortcut(session, appStartMenuPath);
                            if (result.Equals(ActionResult.Failure))
                            {
                                session.Log("Could not create Command Prompt shortcut");
                                // Do not fail if we cannot create shortcut
                                return ActionResult.Success;
                            }
                            break;
                        }
                    default:
                        session.Log(string.Format("Received unknown shortcut, not building: {0}", shortcut));
                        break;

                }
            }
            return ActionResult.Success;
        }

        private static ActionResult PerlCriticShortcut(Session session, string appStartMenuPath)
        {
            session.Log("Installing Perl Critic shortcut");

            string target = Path.Combine(session.CustomActionData["INSTALLDIR"], "bin", "wperl.exe");
            if (!System.IO.File.Exists(target))
            {
                session.Log(string.Format("wperl.exe does not exist in path: {0}", target));
                ActiveState.RollbarHelper.Report(string.Format("wperl.exe does not exist in path: {0}", target));
                return ActionResult.Failure;
            }

            string perlCriticLocation = Path.Combine(session.CustomActionData["INSTALLDIR"], "bin", "perlcritic-gui");
            if (!System.IO.File.Exists(perlCriticLocation))
            {
                session.Log(string.Format("perlcritic-gui does not exist in path: {0}", perlCriticLocation));
                ActiveState.RollbarHelper.Report(string.Format("perlcritic-gui does not exist in path: {0}", perlCriticLocation));
                return ActionResult.Failure;
            }

            if (!Directory.Exists(appStartMenuPath))
                Directory.CreateDirectory(appStartMenuPath);

            string shortcutLocation = Path.Combine(appStartMenuPath, "Perl Critic" + ".lnk");
            WshShell shell = new WshShell();
            IWshShortcut shortcut = (IWshShortcut)shell.CreateShortcut(shortcutLocation);

            shortcut.Description = "Perl Critic";
            shortcut.IconLocation = session.CustomActionData["INSTALLDIR"] + "perl.ico";
            shortcut.TargetPath = target;
            shortcut.Arguments = " -x " + perlCriticLocation;
            shortcut.Save();
            return ActionResult.Success;
        }

        private static ActionResult CmdPromptShortcut(Session session, string appStartMenuPath)
        {
            session.Log("Installing Cmd Prompt shortcut");

            string target = Path.Combine(session.CustomActionData["INSTALLDIR"], "bin", "shell.bat");
            if (!System.IO.File.Exists(target))
            {
                session.Log(string.Format("shell.bat does not exist in path: {0}", target));
                ActiveState.RollbarHelper.Report(string.Format("shell.bat does not exist in path: {0}", target));
                return ActionResult.Failure;
            }

            if (!Directory.Exists(appStartMenuPath))
                Directory.CreateDirectory(appStartMenuPath);

            string shortcutLocation = Path.Combine(appStartMenuPath, "Developer Command Prompt.lnk");
            WshShell shell = new WshShell();
            IWshShortcut shortcut = (IWshShortcut)shell.CreateShortcut(shortcutLocation);

            shortcut.Description = "Developer Command Prompt";
            shortcut.TargetPath = "%comspec%";
            shortcut.Arguments = " /k " + target;
            shortcut.Save();
            return ActionResult.Success;
        }
    }
}
