using System;
using System.Collections.Generic;
using System.Text;
using Microsoft.Deployment.WindowsInstaller;

namespace GetProjectName
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult GetProjectName(Session session)
        {
            session.Log("Attempting to get Project namespace from filename");

            return ActionResult.Success;
        }
    }
}
