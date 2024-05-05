# CSGrader Install Guide
## Installation

The Installation consists of installing the latest version of CSGrader. The latest version of CSGrader has the latest 
version of Typescript grading, RLang grading, and audit logging. After the initial installation, the testing frameworks 
Testthat (https://testthat.r-lib.org/) and Jest (https://jestjs.io/docs/getting-started ) will need to be installed. 
After the project and testing frameworks have been configured, the address for repositories to be graded will need to 
be added to the project configuration.

## New Platform Installation
   
Migrating to a new platform is relatively simple. Assuming a preconfigured container (such as through docker) an 
instance of the CSGrader can be spun up that contains all the necessary prerequisites. Installing Testthat, Jest, or 
another testing framework would not be necessary. Project configuration, such as selecting a repository to pull 
assignments from, may still require configuration.
