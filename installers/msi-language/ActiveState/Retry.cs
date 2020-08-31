using Microsoft.Deployment.WindowsInstaller;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading;

namespace ActiveState
{
    public static class RetryHelper
    {
	public static void RetryOnException(Session session, int times, TimeSpan delay, Action operation)
	{
	    var attempts = 0;
	    do
	    {
		try
		{
		    attempts++;
		    operation();
		    break;  // on success exit the loop
		}
		catch (Exception err)
		{
		    if (attempts == times)
		    {
			throw;
		    }

		    session.Log("Exception caught on attempt #{0}, will retry after {1}, error was: {2}", attempts, delay, err);

		    Thread.Sleep(delay);
		}
	    } while (true);
	}
    }
}
