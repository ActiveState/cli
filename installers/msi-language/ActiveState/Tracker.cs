using GAPixel;
using GoogleAnalyticsTracker.Core;
using GoogleAnalyticsTracker.Core.TrackerParameters;
using GoogleAnalyticsTracker.Simple;
using System;
using System.Collections.Generic;
using System.Threading.Tasks;


namespace ActiveState
{
    public sealed class TrackerSingleton
	{
        private static readonly Lazy<TrackerSingleton> lazy = new Lazy<TrackerSingleton>(() => new TrackerSingleton());
        private static string GoogleAnalyticsUserAgent = "UA-118120158-2";

        private readonly SimpleTracker _tracker;
        private readonly string _cid;

        public static TrackerSingleton Instance {  get { return lazy.Value; } }

        public TrackerSingleton()
		{
            var simpleTrackerEnvironment = new SimpleTrackerEnvironment(Environment.OSVersion.Platform.ToString(),
                Environment.OSVersion.Version.ToString(),
                Environment.OSVersion.VersionString);
            this._tracker = new SimpleTracker(GoogleAnalyticsUserAgent, simpleTrackerEnvironment);
            this._cid = GetInfo.GetUniqueId();
  		}

        public async Task<TrackingResult> TrackEventAsync(string category, string action, string label, long value = 1)
        {
            var eventTrackingParameters = new EventTracking
            {
                Category = category,
                Action = action,
                Label = label,
                Value = value,
            };

            eventTrackingParameters.ClientId = this._cid;

            return await this._tracker.TrackAsync(eventTrackingParameters);
        }

        public void TrackEventInBackground(string category, string action, string label, long value=1)
		{
            Task.Run(() => TrackEventAsync(category, action, label, value));
		}

        public void TrackEventSynchronously(string category, string action, string label, long value=1)
		{
            var t = Task.Run(() => TrackEventAsync(category, action, label, value));
            t.Wait();
        }
    }
};