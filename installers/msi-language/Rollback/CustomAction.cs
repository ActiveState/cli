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
            session.Log("Begin rollback of state tool installation and deploy");

            RollbarHelper.ConfigureRollbarSingleton(session.CustomActionData["COMMIT_ID"]);

            RollbackStateToolInstall(session);
            RollbackDeploy(session);
            
            // This custom action should not abort on failure, just report
            return ActionResult.Success;
        }

        private static void RollbackStateToolInstall(Session session)
        {
            if (session.CustomActionData["STATE_TOOL_INSTALLED"] == "false")
            {
                Status.ProgressBar.StatusMessage(session, "Rolling back State Tool installation");
                // If we installed the state tool then we want to remove it
                // along with any environment entries.
                // We cannot pass data between non-immediate custom actions
                // so we use the known State Tool installation path from the
                // state deploy custom acion.
                string stateToolInstallDir = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "ActiveState", "bin");

                session.Log(string.Format("Attemping to remove State Tool installation directory: {0}", stateToolInstallDir));
                ActionResult result = Remove.Dir(session, stateToolInstallDir);
                if (!result.Equals(ActionResult.Success))
                {
                    string msg = string.Format("Not successful in removing State Tool installation directory, got action result: {0}", result);
                    session.Log(msg);
                    RollbarReport.NonCritical(msg);
                }

                session.Log(string.Format("Removing environment entries containing: {0}", stateToolInstallDir));
                result = Remove.EnvironmentEntries(session, stateToolInstallDir);
                if (!result.Equals(ActionResult.Success))
                {
                    string msg = string.Format("Not successful in removing State Tool environment entries, got action result: {0}", result);
                    session.Log(msg);
                    RollbarReport.NonCritical(msg);

                }
            }

        }

        private static void RollbackDeploy(Session session)
        {
            Status.ProgressBar.StatusMessage(session, "Rolling back language installation");
            ActionResult result = Remove.Dir(session, session.CustomActionData["INSTALLDIR"]);
            if (!result.Equals(ActionResult.Success))
            {
                string msg = string.Format("Not successful in removing deploy directory, got action result: {0}", result);
                session.Log(msg);
                RollbarReport.NonCritical(msg);
            }

            result = Remove.EnvironmentEntries(session, session.CustomActionData["INSTALLDIR"]);
            if (!result.Equals(ActionResult.Success))
            {
                string msg = string.Format("Not successful in removing Deployment environment entries, got action result: {0}", result);
                session.Log(msg);
                RollbarReport.NonCritical(msg);
            }
        }
    }
}
