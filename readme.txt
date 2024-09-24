Commands ----------------------------

- Get the latest version (Backup your config before and add you info to the new config)
"git pull"

- Run NetMess
'go run .'

- Run NetMess and start test instantly
'go run . q' 

- Run NetMess and configure the test to run in reverse mode
'go run . r' 

- Run NetMess and start test instantly in Reverse Mode
'go run . q r' 

--------------------------------------
Example configuration with explanation

{
    "Connections": 1,

    "Names": [
        "Messpunkt-A"
    ],

    "ServerIPList": [
        "192.168.0.209"
    ],

    "ServerPortList": [
        5201
    ],

    "Args" : {
        "Protocol" : "UDP",
        "GetServerData" : true,
        "TestRunimeSeconds" : 12,
        "ReportIntervall": 1,
        "Bandwidth" : "",
        "ParallelStreams" : 0,
        "JSONformat": false
    }
}