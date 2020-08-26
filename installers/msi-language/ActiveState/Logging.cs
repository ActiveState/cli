using Microsoft.Deployment.WindowsInstaller;
using System;
using System.IO;

namespace ActiveState
{
    public sealed class Logging : IDisposable
    {

	private readonly Session _session;
	private readonly string _logFileName;
        private readonly StreamWriter _sw;
        private bool _initialized;

	public Logging(Session session, string installdir)
	{
	    this._session = session;

            this._initialized = false;

            /* My initial idea was to write all the logging in a log file relative to the installation directory.
             * As there is not much time to test if that might intefer with the actual installation, I am pausing that
             * in attempt and instead write to a temporary file */
            if (String.IsNullOrEmpty(installdir))
	    {
                session.Log("cannot create logger, installdir is empty");
                return;
	    }

            /* Note, that this will also include errors from previous installation attempts, as we are just appending.
             * That might be useful in some cases... */
            this._logFileName = Path.Combine(Path.GetTempPath(), "activestate-msi-install.log");

            try
            {
                this._sw = File.AppendText(this._logFileName);
                this._initialized = true;
            }
            catch (Exception err)
            {
                this._session.Log("Failed to open logFile {0}: {1}", this._logFileName, err.ToString());
            }
	}

        // Implement IDisposable.
        // Do not make this method virtual.
        // A derived class should not be able to override this method.
        public void Dispose()
        {
            if (this._sw != null)
            {
                this._sw.Dispose();
            }
        }

        public Session Session()
	{
            return this._session;
	}

        // Use C# destructor syntax for finalization code.
        // This destructor will run only if the Dispose method
        // does not get called.
        // It gives your base class the opportunity to finalize.
        // Do not provide destructors in types derived from this class.
        ~Logging()
        {
            // Do not re-create Dispose clean-up code here.
            Dispose();
        }

        public void Log(string format, params object[] values)
	{
            Log(String.Format(format, values));
	}

        public void Log(string message)
	{
            this._session.Log(message);

            if (this._initialized)
	    {
                this._sw.WriteLine(message);
                this._sw.Flush();
	    }
	}

        public string GetLog()
	{
            if (!File.Exists(this._logFileName))
	    {
                return "Log file did not exist";
	    }
            try
            {
                // open log file with shared access
                using (var fs = File.Open(this._logFileName, FileMode.Open, FileAccess.Read, FileShare.ReadWrite))
		{
                    using (var sr = new StreamReader(fs))
                    {
                        return sr.ReadToEnd();
                    }
		}
            } catch (Exception err)
	    {
                return String.Format("Error reading from log file: {0}", err.ToString());
	    }
	}
    }
}
