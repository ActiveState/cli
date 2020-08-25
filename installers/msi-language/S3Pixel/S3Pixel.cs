using System;
using GAPixel;
using System.Net;
using Microsoft.Deployment.WindowsInstaller;
using System.IO;

namespace S3Pixel
{
    public class CustomActions
    {
        [CustomAction]
        public static ActionResult DownloadPixel(Session session)
        {
            session.Log("Begin download S3 pixel");

            string guid = GetInfo.GetUniqueId(session);
            string pixelURL = string.Format("https://cli-msi.s3.amazonaws.com/pixel.txt?x-referrer={0}", guid);
            
            string timeStamp = DateTime.Now.ToFileTime().ToString();
            string tempDir = Path.Combine(Path.GetTempPath(), timeStamp);
            string pixelFile = Path.Combine(tempDir, "pixel.txt");

            session.Log(string.Format("Downloading S3 pixel from URL: {0}", pixelURL));
            try
            {
                WebClient client = new WebClient();
                client.DownloadFile(pixelURL, pixelFile);
            }
            catch (WebException e)
            {
                string msg = string.Format("Encountered exception downloading S3 pixel file: {0}", e.ToString());
                session.Log(msg);
                RollbarReport.Error(msg);
            }

            return ActionResult.Success;
        }
    }
}
