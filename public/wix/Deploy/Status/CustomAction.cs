using Microsoft.Deployment.WindowsInstaller;

namespace Status
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult ResetProgress(Session session)
        {
            return ProgressBar.Reset(session);
        }
    }

    public class ProgressBar
    {
        public static string ticksString = "4";
        public static int ticks = 10000;

        public static ActionResult Reset(Session session)
        {
            var record = new Record(4);
            record[1] = 0; // "Reset" message 
            record[2] = ProgressBar.ticksString;  // total ticks 
            record[3] = 0; // forward motion 
            record[4] = 0;
            session.Message(InstallMessage.Progress, record);

            return ActionResult.Success;
        }

        public static void StatusMessage(Session session, string status)
        {
            Record record = new Record(3);
            record[1] = "callAddProgressInfo";
            record[2] = status;
            record[3] = "Incrementing tick [1] of [2]";

            session.Message(InstallMessage.ActionStart, record);
        }

        public static MessageResult Increment(Session session, int progressTicks)
        {
            var record = new Record(3);
            record[1] = 2; // "ProgressReport" message 
            record[2] = progressTicks.ToString(); // ticks to increment 
            record[3] = 0; // ignore 
            return session.Message(InstallMessage.Progress, record);
        }
    }
}
