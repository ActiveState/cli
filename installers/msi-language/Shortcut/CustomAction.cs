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
                            return result;
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

            string[] subDirs = Directory.GetDirectories(session.CustomActionData["INSTALLDIR"]);

            string targetDir = "";
            foreach(string dir in subDirs)
            {
                if (dir.EndsWith("bin"))
                {
                    targetDir = dir;
                    break;
                }
            }

            if (targetDir == "")
            {
                session.Log("Could not find binary directory in installation dir");
                return ActionResult.Failure;
            }

            string target = targetDir + @"\wperl.exe";
            if (!System.IO.File.Exists(target))
            {
                session.Log(string.Format("wperl.exe does not exist in path: {0}", targetDir));
                return ActionResult.Failure;
            }

            string perlCriticLocation = targetDir + @"\perlcritic-gui";
            if (!System.IO.File.Exists(perlCriticLocation))
            {
                session.Log(string.Format("perlcritic-gui does not exist in path: {0}", targetDir));
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
