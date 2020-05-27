using System;
using System.Collections.Generic;
using System.Text;
using Microsoft.Deployment.WindowsInstaller;

namespace ResetProgress
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult ResetProgress(Session session)
        {
            var record = new Record(4);
            record[1] = 0; // "Reset" message 
            record[2] = "4";  // total ticks 
            record[3] = 0; // forward motion 
            record[4] = 0;
            session.Message(InstallMessage.Progress, record);

            return ActionResult.Success;
        }
    }
}
