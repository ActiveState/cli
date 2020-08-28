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
        public static LanguagePreset Parse(string ps, ActiveState.Logging log, string installDir, string appStartMenuPath)
        {
            if (ps == "Perl")
            {
                return new PerlPreset(log, installDir, appStartMenuPath);
            }
            return null;
        }
    };

    public class PerlPreset : Preset.LanguagePreset
    {
        private ActiveState.Logging log;
        private string appStartMenuPath;
        private string installPath;

        static string[] PathExtensions =
        {
            ".PL", ".WPL"
        };

        public PerlPreset(ActiveState.Logging log, string installPath, string appStartMenuPath)
        {
            this.log = log;
            this.appStartMenuPath = appStartMenuPath;
            this.installPath = installPath;
        }

        public ActionResult Uninstall()
        {
            log.Log("un-installing perl file associations");
            FileAssociation.EnsureAssociationsDeleted(associations());

            log.Log("removing from PATHEXT");
            var oldPathExt = Environment.GetEnvironmentVariable("PATHEXT", EnvironmentVariableTarget.Machine);
            var newPathExt = String.Join(";", oldPathExt.Split(';').Where(x => !PathExtensions.Contains(x)).ToArray());
            if (newPathExt != oldPathExt)
            {
                log.Log(string.Format("updating PATHEXT to {0}", newPathExt));
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
            log.Log("Install PerlCritic shortcut");
            var result = PerlCriticShortcut();
            if (result.Equals(ActionResult.Failure))
            {
                log.Log("Could not create Perl Critic shortcut");
                // Do not fail if we cannot create shortcut
                return ActionResult.Success;
            }

            log.Log("Install Documentation link");
            DocumentationShortcut();

            log.Log("install cmd-prompt shortcut");
            result = CmdPromptShortcut();
            if (result.Equals(ActionResult.Failure))
            {
                log.Log("Could not create Command Prompt shortcut");
                // Do not fail if we cannot create shortcut
                return ActionResult.Success;
            }

            log.Log("installing perl file associations");
            FileAssociation.EnsureAssociationsSet(associations());

            log.Log("updating PATHEXT");
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
            var installDir = log.Session().CustomActionData["INSTALLDIR"];
            log.Log("Installing Perl Documentation shortcut");

            if (!Directory.Exists(appStartMenuPath))
                Directory.CreateDirectory(appStartMenuPath);

            var shortcutLocation = Path.Combine(appStartMenuPath, "Documentation.url");
            var iconLocation = log.Session().CustomActionData["INSTALLDIR"] + "perl.ico";

            CreateInternetShortcut(shortcutLocation, log.Session().CustomActionData["REL_NOTES"], iconLocation);
        }

        private ActionResult PerlCriticShortcut()
        {
            string shortcutLocation = Path.Combine(appStartMenuPath, "Perl Critic" + ".lnk");

            log.Log("Installing Perl Critic shortcut @ {0}", shortcutLocation);

            string target = Path.Combine(log.Session().CustomActionData["INSTALLDIR"], "bin", "wperl.exe");
            if (!System.IO.File.Exists(target))
            {
                log.Log(string.Format("wperl.exe does not exist in path: {0}", target));
                RollbarReport.Error(string.Format("wperl.exe does not exist in path: {0}", target), this.log);
                return ActionResult.Failure;
            }

            string perlCriticLocation = Path.Combine(log.Session().CustomActionData["INSTALLDIR"], "bin", "perlcritic-gui");
            if (!System.IO.File.Exists(perlCriticLocation))
            {
                log.Log(string.Format("perlcritic-gui does not exist in path: {0}", perlCriticLocation));
                RollbarReport.Error(string.Format("perlcritic-gui does not exist in path: {0}", perlCriticLocation), log);
                return ActionResult.Failure;
            }

            if (!Directory.Exists(appStartMenuPath))
                Directory.CreateDirectory(appStartMenuPath);

            WshShell shell = new WshShell();
            IWshShortcut shortcut = (IWshShortcut)shell.CreateShortcut(shortcutLocation);

            shortcut.Description = "Perl Critic";
            shortcut.IconLocation = log.Session().CustomActionData["INSTALLDIR"] + "perl.ico";
            shortcut.TargetPath = target;
            shortcut.Arguments = " -x " + "\"" + perlCriticLocation + "\"";
            shortcut.Save();
            return ActionResult.Success;
        }

        private ActionResult CmdPromptShortcut()
        {
            string shortcutLocation = Path.Combine(appStartMenuPath, "Developer Command Prompt.lnk");

            log.Log("Installing Cmd Prompt shortcut at {0}", shortcutLocation);

            string target = Path.Combine(log.Session().CustomActionData["INSTALLDIR"], "bin", "shell.bat");
            if (!System.IO.File.Exists(target))
            {
                log.Log(string.Format("shell.bat does not exist in path: {0}", target));
                RollbarReport.Error(string.Format("shell.bat does not exist in path: {0}", target), log);
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
            using (var log = new ActiveState.Logging(session, installDir))
            {

                var preset = Preset.ParsePreset.Parse(presetStr, log, installDir, appStartMenuPath);
                if (preset == null)
                {
                    log.Log("No valid preset set");
                    return ActionResult.Failure;
                }

                try
                {
                    var res = preset.Install();
                    if (res != ActionResult.Success)
                    {
                        RollbarReport.Error(string.Format("unexpected failure in Preset installation"), log);
                    }
                    return res;
                }
                catch (Exception err)
                {
                    RollbarReport.Critical(string.Format("unknown error in language preset: {0}", err), log);
                    return ActionResult.Failure;
                }
            }
        }
    }
}
