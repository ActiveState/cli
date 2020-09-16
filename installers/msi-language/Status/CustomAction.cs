using Microsoft.Deployment.WindowsInstaller;

namespace Status
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult ResetProgress(Session session)
        {
            session.Log("reset progress bar");
            return ProgressBar.Reset(session);
        }

        [CustomAction]
        public static ActionResult SetInstallMode(Session session)
        {
            string mode = "Install";
            session.Log("Setting install mode to {0}", mode);
            session["INSTALL_MODE"] = mode;
            return ActionResult.Success;
        }

        [CustomAction]
        public static ActionResult SetUninstallMode(Session session)
        {
            string mode = "Uninstall";
            session.Log("Setting install mode to {0}", mode);
            session["INSTALL_MODE"] = mode;
            return ActionResult.Success;
        }

        [CustomAction]
        public static ActionResult SetModifyMode(Session session)
        {
            string mode = "Modify";
            session.Log("Setting install mode to {0}", mode);
            session["INSTALL_MODE"] = "Modify";
            return ActionResult.Success;
        }

        [CustomAction]
        public static ActionResult SetRepairMode(Session session)
        {
            string mode = "Repair";
            session.Log("Setting install mode to {0}", mode);
            session["INSTALL_MODE"] = mode;
            return ActionResult.Success;
        }
    }

    public class ProgressBar
    {
        // Set max scale high so we can safely increment by 1 in while loops
        public static string max = "1000";

        public static ActionResult Reset(Session session)
        {
            var record = new Record(4);
            record[1] = 0; // "Reset" message 
            record[2] = ProgressBar.max;  // total ticks 
            record[3] = 0; // forward motion 
            record[4] = 0;
            session.Message(InstallMessage.Progress, record);

            return ActionResult.Success;
        }

        public static MessageResult StatusMessage(Session session, string status)
        {
            Record record = new Record(3);
            record[1] = "callAddProgressInfo";
            record[2] = status;
            record[3] = "Incrementing tick [1] of [2]";

            return session.Message(InstallMessage.ActionStart, record);
        }

        public static MessageResult Increment(Session session, int percentage)
        {
            var record = new Record(3);
            record[1] = 2; // "ProgressReport" message 
            record[2] = percentage.ToString(); // ticks to increment 
            record[3] = 0; // ignore 
            return session.Message(InstallMessage.Progress, record);
        }
    }
}
