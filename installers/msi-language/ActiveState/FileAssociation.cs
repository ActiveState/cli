using Microsoft.Win32;
using System;

namespace ActiveState
{

    public class FileAssociation
    {
        // needed so that Explorer windows get refreshed after the registry is updated
        [System.Runtime.InteropServices.DllImport("Shell32.dll")]
        private static extern int SHChangeNotify(int eventId, int flags, IntPtr item1, IntPtr item2);

        private const int SHCNE_ASSOCCHANGED = 0x8000000;
        private const int SHCNF_FLUSH = 0x1000;

        public string Extension { get; set; }
        public string ProgId { get; set; }
        public string FileTypeDescription { get; set; }
        public string ExecutableFilePath { get; set; }

        public bool SetAssociation()
        {
            bool madeChanges = false;
            madeChanges |= SetKeyDefaultValue(@"Software\Classes\" + Extension, ProgId);
            madeChanges |= SetKeyDefaultValue(@"Software\Classes\" + ProgId, FileTypeDescription);
            madeChanges |= SetKeyDefaultValue($@"Software\Classes\{ProgId}\shell\open\command", "\"" + ExecutableFilePath + "\" \"%1\" %*");
            return madeChanges;
        }

        public bool DeleteAssociation()
        {
            try
            {
                // Do not throw if we the extension value was not set anymore, as it may have been deleted by a different programme.
                Registry.LocalMachine.DeleteValue(@"Software\Classes\" + Extension, false);

                Registry.LocalMachine.DeleteSubKeyTree(@"Software\Classes\" + ProgId, true);
                return true;
            } catch (ArgumentException)
            {
                return false;
            }
        }

        private static bool SetKeyDefaultValue(string keyPath, string value)
        {
            using (var key = Registry.LocalMachine.CreateSubKey(keyPath))
            {
                if (key.GetValue(null) as string != value)
                {
                    key.SetValue(null, value);
                    return true;
                }
            }

            return false;
        }

        public static void EnsureAssociationsSet(params FileAssociation[] associations)
        {
            bool madeChanges = false;
            foreach (var association in associations)
            {
                madeChanges |= association.SetAssociation();
            }

            if (madeChanges)
            {
                SHChangeNotify(SHCNE_ASSOCCHANGED, SHCNF_FLUSH, IntPtr.Zero, IntPtr.Zero);
            }
        }

        public static void EnsureAssociationsDeleted(params FileAssociation[] associations)
        {
            bool madeChanges = false;
            foreach (var association in associations)
            {
                madeChanges |= association.DeleteAssociation();
            }

            if (madeChanges)
            {
                SHChangeNotify(SHCNE_ASSOCCHANGED, SHCNF_FLUSH, IntPtr.Zero, IntPtr.Zero);
            }
        }
    };
}