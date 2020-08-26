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

            if (String.IsNullOrEmpty(installdir))
	    {
                session.Log("cannot create logger, installdir is empty");
                return;
	    }
            this._logFileName = Path.Combine(installdir, "install.log");

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
            this._sw.Dispose();
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
                return File.ReadAllText(this._logFileName);
            } catch (Exception err)
	    {
                return String.Format("Error reading from log file: {0}", err.ToString());
	    }
	}
    }
}
