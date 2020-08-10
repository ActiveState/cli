using System;
using System.Threading.Tasks;
using Rollbar;
using Rollbar.DTOs;
using DeviceId;
using System.Reflection;
using System.Collections.Generic;

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

            RollbarLocator.RollbarInstance
                // minimally required Rollbar configuration:
                .Configure(config)
                ;

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

        public static void Report(string message, IDictionary<string, object> customFields = null )
        {
            RollbarLocator.RollbarInstance.AsBlockingLogger(RollbarTimeout).Error(new GenericException(message), customFields);
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