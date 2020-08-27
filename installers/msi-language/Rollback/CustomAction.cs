using Microsoft.Deployment.WindowsInstaller;
using System;
using System.IO;
using Uninstall;
using ActiveState;

namespace Rollback
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult Rollback(Session session)
        {
            var installDir = session.CustomActionData["INSTALLDIR"];
            RollbarHelper.ConfigureRollbarSingleton(session.CustomActionData["COMMIT_ID"]);

            using (var log = new ActiveState.Logging(session, installDir))
            {
                log.Log("Begin rollback of state tool installation and deploy");

                RollbackStateToolInstall(log);
                RollbackDeploy(log);

                // This custom action should not abort on failure, just report
                return ActionResult.Success;
            }
        }

        private static void RollbackStateToolInstall(ActiveState.Logging log)
        {
            if (log.Session().CustomActionData["STATE_TOOL_INSTALLED"] == "false")
            {
                Status.ProgressBar.StatusMessage(log.Session(), "Rolling back State Tool installation");
                // If we installed the state tool then we want to remove it
                // along with any environment entries.
                // We cannot pass data between non-immediate custom actions
                // so we use the known State Tool installation path from the
                // state deploy custom acion.
                string stateToolInstallDir = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "ActiveState", "bin");

                log.Log(string.Format("Attemping to remove State Tool installation directory: {0}", stateToolInstallDir));
                ActionResult result = Remove.Dir(log, stateToolInstallDir);
                if (!result.Equals(ActionResult.Success))
                {
                    string msg = string.Format("Not successful in removing State Tool installation directory, got action result: {0}", result);
                    log.Log(msg);
                    RollbarReport.Error(msg, log);
                }

                log.Log(string.Format("Removing environment entries containing: {0}", stateToolInstallDir));
                result = Remove.EnvironmentEntries(log, stateToolInstallDir);
                if (!result.Equals(ActionResult.Success))
                {
                    string msg = string.Format("Not successful in removing State Tool environment entries, got action result: {0}", result);
                    log.Log(msg);
                    RollbarReport.Error(msg, log);

                }
            }

        }

        private static void RollbackDeploy(ActiveState.Logging log)
        {
            Status.ProgressBar.StatusMessage(log.Session(), "Rolling back language installation");
            ActionResult result = Remove.Dir(log, log.Session().CustomActionData["INSTALLDIR"]);
            if (!result.Equals(ActionResult.Success))
            {
                string msg = string.Format("Not successful in removing deploy directory, got action result: {0}", result);
                log.Log(msg);
                RollbarReport.Error(msg, log);
            }

            result = Remove.EnvironmentEntries(log, log.Session().CustomActionData["INSTALLDIR"]);
            if (!result.Equals(ActionResult.Success))
            {
                string msg = string.Format("Not successful in removing Deployment environment entries, got action result: {0}", result);
                log.Log(msg);
                RollbarReport.Error(msg, log);
            }
        }
    }
}
