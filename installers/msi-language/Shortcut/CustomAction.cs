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
            string shortcutData = session.CustomActionData["SHORTCUTS"];
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
                        ActionResult result = PerlCriticShortcut(session);
                        if (result.Equals(ActionResult.Failure))
                        {
                            session.Log("Could not create Perl Critic shortcut");
                            // Do not fail if we cannot create shortcut
                            return ActionResult.Success;
                        }
                        break;
                    default:
                        session.Log(string.Format("Recieved unknown shortcut, not building: {0}", shortcut));
                        break;

                }
            }
            return ActionResult.Success;
        }

        private static ActionResult PerlCriticShortcut(Session session)
        {
            session.Log("Installing Perl Critic shortcut");

            string target = session.CustomActionData["INSTALLDIR"] + @"\bin\wperl.exe";
            if (!System.IO.File.Exists(target))
            {
                session.Log(string.Format("wperl.exe does not exist in path: {0}", target));
                return ActionResult.Failure;
            }

            string perlCriticLocation = session.CustomActionData["INSTALLDIR"]+ @"\bin\perlcritic-gui";
            if (!System.IO.File.Exists(perlCriticLocation))
            {
                session.Log(string.Format("perlcritic-gui does not exist in path: {0}", perlCriticLocation));
                return ActionResult.Failure;
            }

            string projectName = session.CustomActionData["PROJECT_OWNER_AND_NAME"];
            string shortcutDir = projectName.Substring(projectName.IndexOf("/")+1);

            string commonStartMenuPath = Environment.GetFolderPath(Environment.SpecialFolder.CommonStartMenu);
            string appStartMenuPath = Path.Combine(commonStartMenuPath, "Programs", shortcutDir);

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
    }
}
