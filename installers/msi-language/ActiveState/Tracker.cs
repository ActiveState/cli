using GAPixel;
using GoogleAnalyticsTracker.Core;
using GoogleAnalyticsTracker.Core.TrackerParameters;
using GoogleAnalyticsTracker.Simple;
using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Net;
using System.Threading.Tasks;


namespace ActiveState
{
    public sealed class TrackerSingleton
    {
        private static readonly Lazy<TrackerSingleton> lazy = new Lazy<TrackerSingleton>(() => new TrackerSingleton());
        private static string GoogleAnalyticsUserAgent = "UA-118120158-2";

        private readonly SimpleTracker _tracker;
        private readonly string _cid;

        public static TrackerSingleton Instance { get { return lazy.Value; } }

        public TrackerSingleton()
        {
            var simpleTrackerEnvironment = new SimpleTrackerEnvironment(Environment.OSVersion.Platform.ToString(),
                Environment.OSVersion.Version.ToString(),
                Environment.OSVersion.VersionString);
            this._tracker = new SimpleTracker(GoogleAnalyticsUserAgent, simpleTrackerEnvironment);
            this._cid = GetInfo.GetUniqueId();
        }

        private async Task<TrackingResult> TrackEventAsync(Session session, string category, string action, string label, GACustomDimensions gd, long value = 1)
        {
            session.Log("Sending GA Event");
            var eventTrackingParameters = new EventTracking
            {
                Category = category,
                Action = action,
                Label = label,
                Value = value,
            };

            eventTrackingParameters.ClientId = this._cid;
            eventTrackingParameters.SetCustomDimensions(new System.Collections.Generic.Dictionary<int, string> {
                { 1, gd.productVersion },
                { 2, gd.sessionID },
                { 3, gd.uiLevel },
                { 4, gd.installMode },
            });

            return await this._tracker.TrackAsync(eventTrackingParameters);
        }

        public async Task TrackS3Event(Session session, string sessionID, string category, string action, string label)
        {
            string pixelURL = string.Format(
                "https://cli-msi.s3.amazonaws.com/pixel.txt?x-referrer={0}&x-session={1}&x-event={2}&x-event-category={3}&x-event-value={4}",
                this._cid, sessionID, action, category, label
            );
            session.Log(string.Format("Downloading S3 pixel from URL: {0}", pixelURL));
            try
            {
                await Task.Run(() =>
                {
                    // retry up to 3 times to download the S3 pixel
                    RetryHelper.RetryOnException(session, 3, TimeSpan.FromSeconds(1), () =>
                    {
                        var client = new TimeoutWebClient();
                        // try to complete an s3 tracking event in seven seconds or less.
                        client.Timeout = 7 * 1000;
                        var res = client.DownloadString(pixelURL);
                        session.Log("Received response {0}", res);
                    });
                });
            }
            catch (Exception e)
            {
                string msg = string.Format("Encountered exception downloading S3 pixel file: {0}", e.ToString());
                session.Log(msg);
                RollbarReport.Error(msg, session);
            }
            session.Log("Successfully downloaded S3 pixel string");
        }

        internal class GACustomDimensions
        {
            public string productVersion;
            public string sessionID;
            public string uiLevel;
            public string installMode;

            public GACustomDimensions(Session session, string cid)
            {
                this.productVersion = getValueFromSession(session, "ProductVersion", "PRODUCT_VERSION");
                var msiLogFileName = getValueFromSession(session, "MsiLogFileLocation", "MsiLogFileLocation");
                this.sessionID = computeSessionID(cid, msiLogFileName);
                this.uiLevel = getValueFromSession(session, "UILevel", "UI_LEVEL");
                this.installMode = getValueFromSession(session, "INSTALL_MODE", "INSTALL_MODE");
            }

            private string computeSessionID(string cid, string msiLogFileName)
            {
                using (var md5 = System.Security.Cryptography.MD5.Create())
                {
                    byte[] inputBytes = System.Text.Encoding.ASCII.GetBytes(cid + msiLogFileName);
                    byte[] hashBytes = md5.ComputeHash(inputBytes);

                    return new Guid(hashBytes).ToString();
                }
            }


            private string getValueFromSession(Session session, string key1, string key2)
            {
                try {
                    if (!session.GetMode(InstallRunMode.Scheduled))
                    {
                        return session[key1];
                    }
                    if (session.CustomActionData.ContainsKey(key2)) {
                        return session.CustomActionData[key2];
                    }
                    return "";
                } catch (Exception err) {
                    session.Log("Error getting value for key {0} from session object: {1}", key1, err);
                    return "error";
                }
            }

        }

        /// <summary>
        /// Sends a GA event in background (fires and forgets)
        /// </summary>
        /// <description>
        /// The event can fail to be send if the main process gets cancelled before the task finishes.
        /// Use the synchronous version of this command in that case.
        /// </description>
        public void TrackEventInBackground(Session session, string msiLogFileName, string category, string action, string label, long value = 1)
        {
            var cd = new GACustomDimensions(session, this._cid);
            session.Log("Sending background event {0}/{1}/{2} for cid={3} (custom dimension 1: {4})", category, action, label, this._cid, cd.productVersion);
            Task.WhenAll(
                TrackEventAsync(session, category, action, label, cd, value),
                TrackS3Event(session, cd.sessionID, category, action, label)
            );
        }

        /// <summary>
        /// Sends a GA event and waits for the request to complete.
        /// </summary>
        public void TrackEventSynchronously(Session session, string category, string action, string label, long value = 1)
        {
            var cd = new GACustomDimensions(session, this._cid);

            if (cd.productVersion == "0.0.0")
            {
                session.Log("Not tracking events when version is 0.0.0");
                return;
            }

            session.Log("Sending event {0}/{1}/{2} for cid={3} (custom dimension 1: {4})", category, action, label, this._cid, cd.productVersion);
            var t = Task.WhenAll(
                TrackEventAsync(session, category, action, label, cd, value),
                TrackS3Event(session, cd.sessionID, category, action, label)
            );
            var completed = t.Wait(TimeSpan.FromSeconds(15));
            if (!completed)
            {
                session.Log("Abandoning tracking event task after timeout.");
            }
        }
    }
};
