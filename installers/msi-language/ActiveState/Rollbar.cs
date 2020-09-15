﻿using System;
using System.Threading.Tasks;
using Rollbar;
using Rollbar.DTOs;
using DeviceId;
using System.Reflection;
using System.Collections.Generic;
using Microsoft.Deployment.WindowsInstaller;
using ActiveState;

namespace ActiveState
{
    /// <summary>
    /// Class RollbarHelper.
    /// A utility class aiding in Rollbar SDK usage.
    /// </summary>
    public static class RollbarHelper
    {
        public static readonly TimeSpan RollbarTimeout = TimeSpan.FromSeconds(10);

        /// <summary>
        /// Configures the Rollbar singleton-like notifier.
        /// </summary>
        public static void ConfigureRollbarSingleton(string codeVersion)
        {
            const string rollbarAccessToken = "72be571d37fa4e54ac487f7d8d78a83f";
            const string rollbarEnvironment = "production";

            var config = new RollbarConfig // minimally required Rollbar configuration
            {
                AccessToken = rollbarAccessToken,
                Environment = rollbarEnvironment,
                Transform = payload =>
                {
                    payload.Data.CodeVersion = codeVersion;
                }
            };
            RollbarLocator.RollbarInstance.Configure(config);

            string deviceId = new DeviceIdBuilder()
                .AddMachineName()
                .AddMacAddress()
                .AddProcessorId()
                .AddMotherboardSerialNumber()
                .ToString();
            SetRollbarReportingUser(deviceId, Environment.UserName);

            AppDomain.CurrentDomain.UnhandledException += (sender, args) =>
            {
                var newExc = new System.Exception("CurrentDomainOnUnhandledException", args.ExceptionObject as System.Exception);
                RollbarLocator.RollbarInstance.AsBlockingLogger(RollbarTimeout).Critical(newExc);
            };

            TaskScheduler.UnobservedTaskException += (sender, args) =>
            {
                var newExc = new ApplicationException("TaskSchedulerOnUnobservedTaskException", args.Exception);
                RollbarLocator.RollbarInstance.AsBlockingLogger(RollbarTimeout).Critical(newExc);
            };
        }

        /// <summary>
        /// Sets the rollbar reporting user.
        /// </summary>
        /// <param name="id">The identifier.</param>
        /// <param name="email">The email.</param>
        /// <param name="userName">Name of the user.</param>
        private static void SetRollbarReportingUser(string id, string userName)
        {
            Person person = new Person(id);
            person.UserName = userName;
            RollbarLocator.RollbarInstance.Config.Person = person;
        }
    }
}

public class RollbarReport
{
    private enum Level
    {
        Critical,
        Error
    }

    private static bool criticalReported;
    private static object syncLock = new object();

    public static readonly TimeSpan RollbarTimeout = TimeSpan.FromSeconds(10);

    public static void Critical(string message, Session session, IDictionary<string, object> customFields = null)
    {
        Report(Level.Critical, message, session, customFields);
    }

    public static void Error(string message, Session session, IDictionary<string, object> customFields = null)
    {
        Report(Level.Error, message, session, customFields);
    }

    private static void Report(Level level, string message, Session session, IDictionary<string, object> customFields = null)
    {
        lock (syncLock)
        {
            // create a custom fields dictionary if necessary
            if (customFields == null)
		    {
                customFields = new Dictionary<string, object>();
		    }
            customFields.Add("log", Logging.GetLog(session));
            string properties = Logging.GetProperties(session);
            if (properties != "")
            {
                customFields.Add("properties", properties);
            }
            var userEnvironment = Logging.GetUserEnvironment(session);
            if (userEnvironment != null)
            {
                customFields.Add("userEnvironment", userEnvironment);
            }
            string installMode = Logging.GetInstallMode(session);
            if (installMode != "")
            {
                customFields.Add("installMode", installMode);
            }
            string country= Logging.GetCountry(session);
            if (country != "")
            {
                customFields.Add("country", country);
            }

            if (!criticalReported)
            {
                try
                {
                    if (level == Level.Critical)
                    {
                        criticalReported = true;
                        RollbarLocator.RollbarInstance.AsBlockingLogger(RollbarTimeout).Critical(new GenericException(message), customFields);
                    }
                    else
                    {
                        RollbarLocator.RollbarInstance.AsBlockingLogger(RollbarTimeout).Error(new GenericException(message), customFields);
                    }
                } catch (System.Exception e)
                {

                    string msiLogFileName = "";
                    string productVersion = "";
                    if (session.GetMode(InstallRunMode.Scheduled))
                    {
                        if (session.CustomActionData.ContainsKey("MsiLogFileLocation"))
                        {
                            msiLogFileName = session.CustomActionData["MsiLogFileLocation"];
                        }
                        if (session.CustomActionData.ContainsKey("PRODUCT_VERSION"))
                        {
                            productVersion = session.CustomActionData["PRODUCT_VERSION"];
                        }
                    }
                    else if (!session.GetMode(InstallRunMode.Scheduled))
                    {
                        msiLogFileName = session["MsiLogFileLocation"];
                        productVersion = session["ProductVersion"];
                    }

                    TrackerSingleton.Instance.TrackEventSynchronously(session, msiLogFileName, "error", "rollbar", "", productVersion);
                    session.Log("Logging to rollbar failed with error: {0}", e);
                }
            }
        }
    }

}

public class GenericException : System.Exception
{
    public GenericException(string message) : base(message)
    {
        // This isn't working (stack is still empty) - leaving it here so we can iterate later
        var stackTraceField = typeof(GenericException).BaseType
            .GetField("_stackTraceString", BindingFlags.Instance | BindingFlags.NonPublic);

        stackTraceField.SetValue(this, Environment.StackTrace);
    }
}
