using ActiveState;
using System.IO;
using IWshRuntimeLibrary;
using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Linq;

namespace Preset
{
    public interface LanguagePreset
    {
        ActionResult Install();
        ActionResult Uninstall();
    }

    public class ParsePreset
    {
        public static LanguagePreset Parse(string ps, Session session, string installDir, string appStartMenuPath)
        {
            if (ps == "Perl")
            {
                return new PerlPreset(session, installDir, appStartMenuPath);
            }
            return null;
        }
    };

    public class PerlPreset : Preset.LanguagePreset
    {
        private Session session;
        private string appStartMenuPath;
        private string installPath;

        static string[] PathExtensions =
        {
            ".PL", ".WPL"
        };

        public PerlPreset(Session session, string installPath, string appStartMenuPath)
        {
            this.session = session;
            this.appStartMenuPath = appStartMenuPath;
            this.installPath = installPath;
        }

        public ActionResult Uninstall()
        {
            session.Log("un-installing perl file associations");
            FileAssociation.EnsureAssociationsDeleted(associations());

            session.Log("removing from PATHEXT");
            var oldPathExt = Environment.GetEnvironmentVariable("PATHEXT", EnvironmentVariableTarget.Machine);
            var newPathExt = String.Join(";", oldPathExt.Split(';').Where(x => !PathExtensions.Contains(x)).ToArray());
            if (newPathExt != oldPathExt)
            {
                session.Log(string.Format("updating PATHEXT to {0}", newPathExt));
                Environment.SetEnvironmentVariable("PATHEXT", newPathExt, EnvironmentVariableTarget.Machine);
            }

            return ActionResult.Success;
        }

        private FileAssociation[] associations()
        {
            return new FileAssociation[] {
                new FileAssociation {
                    Extension = ".pl",
                    ProgId = "Perl.Document",
                    FileTypeDescription = "Perl Document",
                    ExecutableFilePath = Path.Combine(this.installPath, "bin", "perl.exe"),
                },
                new FileAssociation {
                    Extension = ".wpl",
                    ProgId = "WPerl.Document",
                    FileTypeDescription = "WPerl Document",
                    ExecutableFilePath = Path.Combine(this.installPath, "bin", "wperl.exe"),
                }
            };
        }

        public ActionResult Install()
        {
            session.Log("Install PerlCritic shortcut");
            var result = PerlCriticShortcut();
            if (result.Equals(ActionResult.Failure))
            {
                session.Log("Could not create Perl Critic shortcut");
                // Do not fail if we cannot create shortcut
                return ActionResult.Success;
            }

            session.Log("Install Documentation link");
            DocumentationShortcut();

            session.Log("install cmd-prompt shortcut");
            result = CmdPromptShortcut();
            if (result.Equals(ActionResult.Failure))
            {
                session.Log("Could not create Command Prompt shortcut");
                // Do not fail if we cannot create shortcut
                return ActionResult.Success;
            }

            session.Log("installing perl file associations");
            FileAssociation.EnsureAssociationsSet(associations());

            session.Log("updating PATHEXT");
            var oldPathExt = Environment.GetEnvironmentVariable("PATHEXT", EnvironmentVariableTarget.Machine);
            var exts = String.Join(";", oldPathExt.Split(';').Concat(PathExtensions).Distinct().ToArray());
            if (exts != oldPathExt)
            {
                Environment.SetEnvironmentVariable("PATHEXT", exts, EnvironmentVariableTarget.Machine);
            }

            return ActionResult.Success;
        }

        private void CreateInternetShortcut(string path, string url, string icon)
        {
            using (StreamWriter writer = new StreamWriter(path))
            {
                writer.WriteLine("[InternetShortcut]");
                writer.WriteLine("URL=" + url);
                writer.WriteLine("IconIndex=0");
                writer.WriteLine("IconFile=" + icon);
            }
        }

        private void DocumentationShortcut()
        {
            var installDir = session.CustomActionData["INSTALLDIR"];
            session.Log("Installing Perl Documentation shortcut");

            if (!Directory.Exists(appStartMenuPath))
                Directory.CreateDirectory(appStartMenuPath);

            var shortcutLocation = Path.Combine(appStartMenuPath, "Documentation.url");
            var iconLocation = session.CustomActionData["INSTALLDIR"] + "perl.ico";

            CreateInternetShortcut(shortcutLocation, session.CustomActionData["REL_NOTES"], iconLocation);
        }

        private ActionResult PerlCriticShortcut()
        {
            string shortcutLocation = Path.Combine(appStartMenuPath, "Perl Critic" + ".lnk");

            session.Log("Installing Perl Critic shortcut @ {0}", shortcutLocation);

            string target = Path.Combine(session.CustomActionData["INSTALLDIR"], "bin", "wperl.exe");
            if (!System.IO.File.Exists(target))
            {
                session.Log(string.Format("wperl.exe does not exist in path: {0}", target));
                RollbarReport.Error(string.Format("wperl.exe does not exist in path: {0}", target), session);
                return ActionResult.Failure;
            }

            string perlCriticLocation = Path.Combine(session.CustomActionData["INSTALLDIR"], "bin", "perlcritic-gui");
            if (!System.IO.File.Exists(perlCriticLocation))
            {
                session.Log(string.Format("perlcritic-gui does not exist in path: {0}", perlCriticLocation));
                RollbarReport.Error(string.Format("perlcritic-gui does not exist in path: {0}", perlCriticLocation), session);
                return ActionResult.Failure;
            }

            if (!Directory.Exists(appStartMenuPath))
                Directory.CreateDirectory(appStartMenuPath);

            WshShell shell = new WshShell();
            IWshShortcut shortcut = (IWshShortcut)shell.CreateShortcut(shortcutLocation);

            shortcut.Description = "Perl Critic";
            shortcut.IconLocation = session.CustomActionData["INSTALLDIR"] + "perl.ico";
            shortcut.TargetPath = target;
            shortcut.Arguments = " -x " + "\"" + perlCriticLocation + "\"";
            shortcut.Save();
            return ActionResult.Success;
        }

        private ActionResult CmdPromptShortcut()
        {
            string shortcutLocation = Path.Combine(appStartMenuPath, "Developer Command Prompt.lnk");

            session.Log("Installing Cmd Prompt shortcut at {0}", shortcutLocation);

            string target = Path.Combine(session.CustomActionData["INSTALLDIR"], "bin", "shell.bat");
            if (!System.IO.File.Exists(target))
            {
                session.Log(string.Format("shell.bat does not exist in path: {0}", target));
                RollbarReport.Error(string.Format("shell.bat does not exist in path: {0}", target), session);
                return ActionResult.Failure;
            }

            if (!Directory.Exists(appStartMenuPath))
                Directory.CreateDirectory(appStartMenuPath);

            WshShell shell = new WshShell();
            IWshShortcut shortcut = (IWshShortcut)shell.CreateShortcut(shortcutLocation);

            shortcut.Description = "Developer Command Prompt";
            shortcut.TargetPath = "%comspec%";
            shortcut.Arguments = " /k " + "\"" + target + "\"";
            shortcut.Save();
            return ActionResult.Success;
        }

    }

    public class CustomActions
    {
        [CustomAction]
        public static ActionResult InstallPreset(Session session)
        {
            string presetStr = session.CustomActionData["PRESET"];
            string appStartMenuPath = session.CustomActionData["APP_START_MENU_PATH"];
            string installDir = session.CustomActionData["INSTALLDIR"];

            RollbarHelper.ConfigureRollbarSingleton(session.CustomActionData["COMMIT_ID"]);

            var preset = Preset.ParsePreset.Parse(presetStr, session, installDir, appStartMenuPath);
            if (preset == null)
            {
                session.Log("No valid preset set");
                return ActionResult.Failure;
            }

            try
            {
                var res = preset.Install();
                if (res != ActionResult.Success)
                {
                    RollbarReport.Error(string.Format("unexpected failure in Preset installation"), session);
                }
                return res;
            }
            catch (Exception err)
            {
                RollbarReport.Critical(string.Format("unknown error in language preset: {0}", err), session);
                return ActionResult.Failure;
            }
        }
    }
}
