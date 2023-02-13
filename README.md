# Example service monitoring
This application monitors the availability of a specific set of services which is given in the file 'sites.txt'.
In period of N-seconds, it checks the availability and accessing time of the service.
Normal users can get service's status and force to update this information.
In addition, administrators can track statistics of requests from users for all services.

# Components
There are 3 service:
- monitor: an exportable component, implements users handlers, caches.
- tracker: an internal component, using embbeded SQL 'genji' to store the user's request information.
- sampler: an internal component, updates sites' status.

# Building
Integrated scripts will build components into the folder ```bin```:
- Runs the script ```build.sh``` to build all components
- To build seperate components, go to component's folder and run ```build.sh```.

# Running
- Runs the script ```.\run.sh``` to build all components with corresponding log files.
Default port is 8090, 8091, 8092. To use another range ports, running ```.\run.sh``` with extra
port parameter, ex: ```.\run.sh 9000``` will use ports 9000 for service 'monitor', 9001 for service 'tracker', 9002 for service 'sampler'.

- To run seperate components, go to component's folder and run ```run.sh```.

- To run a component with specific paramenters, go to the folder ```bin``` and run it directly, use parameter ```-h``` for help.

# API
## Admin
All admin's requests requires ```admin_token``` value in the header must equal to the token given in the monitor's argument ```-a``` or ```--admin```.

- /admin_query_all?from={{from_value}}&to={{to_value}}

  Count all number of users' requests in a range of time.
  - Query params:
    - from: the start of time range value in the unix-epoch second format.
    - to: the end of time range value in the unix-epoch second format, omit this parameter to use the current time value.
  - Respond: the number of requests if success.


- /admin_query_one?from={{from_value}}&to={{to_value}}&user={{user_value}}

  Count the number of requests of a specific user in a range of time.
  - Query params:
    - from: the start of time range value in the unix-epoch second format.
    - to: the end of time range value in the unix-epoch second format, omit this parameter to use the current time value.
    - user: the id of user.
  - Respond: the number of requests if success.

## Normal user
All requests of normal user requires ```user_id``` value in the header.
- /check?target={{target_value_1}}&&target={{target_value_2}}
  
  Get the status of target sites.
  - Query params:
    - target: the address of sites.
  - Respond: the JSON object contains status of sites, ex:
    ```
    {
        "google.co.jp": {
            "availability": true,
            "access_time": 56059100
        },
        "reddit.com": {
            "availability": true,
            "access_time": 87297100
        }
    }
    ```
    Note: the unit of the field ```access_time``` is nanoseconds.

- /force?target={{target_value}}

  Force to update status of a target site.
  - Query params:
    - target: the address of a site.
  - Respond: status code is 200 if success.

- /min

  Get the current fastest site.
  - Respond: the JSON object contains information of the current fastest site if available, ex:
    ```
    {
        "address": "live.com",
        "availability": true,
        "access_time": 51759000
    }
    ```
- /max

  Get the current slowest site.
  - Respond: the JSON object contains information of the current slowest site if available, ex:
    ```
    {
        "address": "jd.com",
        "availability": true,
        "access_time": 497269300
    }
    ```

