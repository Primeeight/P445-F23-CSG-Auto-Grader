@startuml
package "R Docker Container"{
"R Language" - [R testing framework]
}
[R testing framework] ..> "HTTP Request": publish results
[R testing framework] <.. "HTTP Request": accept assignments

package "Typescript Docker Container"{
"Typescript Language" - [Typescript testing framework]

}
[Typescript testing framework] ..> "HTTP Request": publish results
[Typescript testing framework] <.. "HTTP Request": accept assignments
"Audit Logging" - [R testing framework]
"Audit Logging" - [Typescript testing framework]

cloud {
[Program Grader]
}
"Audit Logging" - [Program Grader]
"HTTP Request" - [Program Grader]
node "CSG Grader" {
[CSG Grader Frontend]
}
"HTTP Request" -- [CSG Grader Frontend]
@enduml