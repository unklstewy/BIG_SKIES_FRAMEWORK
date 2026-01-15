The BIG SKIES FRAMEWORK is intended to be a plugin extensible framework that will utilze plugins, as containers, to extend coordinated services, ui-extension, database management, health monitoring, application control etc. in order to support telescope operations for both terrestrial and astronomy operations. 
Examples of potential plugins: 
    Terrestrial: 
        - ADS-B Data Source to Telescope Coordinates for imaging aircraft in flight, approaching, departing, etc.
        - Wildlife Tracker, perhaps vision based ai to identify a wild life target, track it visually, further target classification: distinguishing individual targets of type by markings, etc. 
    Astronomy:  
        - DeepSky@Home - Distributed Deep Sky Survey Workloads ala Seti@Home, Folding@Home, where workload chunks are distributed to participating observers.
        - Astro Photography Spatial Viewer for VR, examine your astrophtography from within a virual sky dome, ideally would integrate with the imaging results from the DeepSky@Home project. 

#REPOSITORY: Use to match go mod
git@github.com:unklstewy/BIG_SKIES_FRAMEWORK.git



# GOALS
1. CREATE AN BACKEND FRAMEWORK 'BIG SKIES' THAT IS EXTENSIBLE BY PLUGINS
2. FRAMEWORK MUST ADHERE TO MODERN SECURITY MODELS
3. FRAMEWORK MUST SUPPORT mTLS 
4. FRAMEWORK MUST SUPPORT DOCKER CONTAINERS AS PLUGIN EXTENSIONS AS DESRIBED IN /Users/unklstewy/Development/BIG_SKIES/docs/architecture/big_skies_architecture_gojs.json


# RULES
FRAMEWORK MUST FOLLOW MODERING PWA SECURITY FOCUSED DESIGN
FRAMEWORK MUST SUPPORT ROLE BASED AUTHORITY MODEL

FRAMEWORK MUST UTILIZE AN OPENSOURCE DATABASE WITH STRONG SECURITY MODELS (POSTGRES CONTAINER)
FRAMEWORK MUST BE WRITTEN IN GO LANGUAGE. (Installed)
FRAMEWORK MUST ALSO SUPPORT ALPACA ASCOM SIMULATOR FOR TESTING (NEEDS TO BE INSTALLED)
FRAMEWORK SUPPORRTING APPLICATIONS MUST BE CONTAINERIZED
FRAMEWORK SOURCE CODE MUST BE HEAVILY COMMENTED TO PROVIDE FUTURE INSIGHTS INTO EACH FUNCTION, METHOD, INTERFACE, STRUCT, ETC.
 


#REFERENCES
Alpaca ASCOM Interface repos: https://github.com/ASCOMInitiative

#CI/CD
DEVELOPMENT SHOULD MAKE USE OF CI/CD TOOLS IN GITHUB
DEVELOPMENT SHOULD TRACK PROGRESS WITH GITHUB ISSUES TRACKER
DEVELOPMENT SHOULD RECORD DOCUMENTATION IN GITHUB WIKI ENTRIES
DEVELOPMENT CONTRIBUTIONS POLICY NEEDS TO BE DEFINED - FOLLOW STANDARD BEST PRACTICES
