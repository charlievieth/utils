#using <System.dll>

using namespace System;
using namespace System::Diagnostics;

Int64 DisplayTimerProperties()
{
   // Display the timer frequency and resolution.
   if ( Stopwatch::IsHighResolution )
   {
      return 1;
   }
   return 0;
}
