# Matillion Task History

This package can be user to interact with Matillion API and retrieve the history of tasks run. 

More specifically it can be used to
- retrieve the list of groups 
- retrieve the list of projects
- retrieve the tasks history from any point in the past, the logic will handle retrieving it in small batches.

This package is an helper and it is meant to have a tool built around it to manage the extracted information.

### Please note
Matillion won't expose a task in the history API until its execution is completed.
This can have implication if you are reading the history incrementally using the "last start date" as a marker that you previously saved somewhere.  
For this reason when querying the history you have the option to use the end date as a filter.  
using the end date as a marker you are sure that you are not leaving behind/missing tasks. 
