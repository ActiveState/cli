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

        public async Task<TrackingResult> TrackEventAsync(Session session, string sessionID, string sessionID, string category, string action, string label, string msiVersion, long value = 1)
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
                { 1, productVersion },
                { 2, sessionID },
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
                var client = new TimeoutWebClient();
                // never attempt to send pixel for more than 15 seconds, as it blocks the entire MSI
                client.Timeout = 15 * 1000;
                await client.DownloadStringTaskAsync(pixelURL);
            }
            catch (WebException e)
            {
                string msg = string.Format("Encountered exception downloading S3 pixel file: {0}", e.ToString());
                session.Log(msg);
                RollbarReport.Error(msg, session);
            }

            session.Log("Successfully downloaded S3 pixel string");
        }

        private string computeSessionID(string msiLogFileName)
        {
            using (var md5 = System.Security.Cryptography.MD5.Create())
            {
                byte[] inputBytes = System.Text.Encoding.ASCII.GetBytes(this._cid + msiLogFileName);
                byte[] hashBytes = md5.ComputeHash(inputBytes);

                return new Guid(hashBytes).ToString();
            }
        }

        /// <summary>
        /// Sends a GA event in background (fires and forgets)
        /// </summary>
        /// <description>
        /// The event can fail to be send if the main process gets cancelled before the task finishes.
        /// Use the synchronous version of this command in that case.
        /// </description>
        public void TrackEventInBackground(Session session, string msiLogFileName, string category, string action, string label, string productVersion, long value = 1)
        {
            var pid = System.Diagnostics.Process.GetCurrentProcess().Id;

            var sessionID = computeSessionID(msiLogFileName);
            session.Log("Sending background event {0}/{1}/{2} for cid={3} (custom dimension 1: {4}, pid={5})", category, action, label, this._cid, productVersion, pid);
            Task.WhenAll(
                TrackEventAsync(session, sessionID, category, action, label, productVersion, value),
                TrackS3Event(session, sessionID, category, action, label)
            );
        }

        /// <summary>
        /// Sends a GA event and waits for the request to complete.
        /// </summary>
        public void TrackEventSynchronously(Session session, string msiLogFileName, string category, string action, string label, string productVersion, long value = 1)
        {
            if (productVersion == "0.0.0")
            {
                session.Log("Not tracking events when version is 0.0.0");
                return;
            }

            var pid = System.Diagnostics.Process.GetCurrentProcess().Id;

            session.Log("Sending event {0}/{1}/{2} for cid={3} (custom dimension 1: {4}, pid={5})", category, action, label, this._cid, productVersion, pid);
            var sessionID = computeSessionID(msiLogFileName);
            var t = Task.WhenAll(
                TrackEventAsync(session, sessionID, category, action, label, productVersion, value),
                TrackS3Event(session, sessionID, category, action, label)
            );
            var completed = t.Wait(TimeSpan.FromSeconds(15));
            if (!completed)
            {
                session.Log("Abandoning tracking event task after timeout.");
            }
        }
    }
};
