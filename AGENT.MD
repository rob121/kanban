This app is a golang based Kanban Board App

Main Packages to use:

Gorm - gorm.io
https://github.com/markbates/goth - auth, google auth to start
https://github.com/rob121/vhelp - for configuration

CSS: 
Bootstrap 5

Javascript:
https://turbo.hotwired.dev/

Database: sqlite Optional: mysql,postgresql

Design Brief - A kanban board with usability, UI/UX, and light/dark mode exists. Cards must be draggable/dropable between categories.

The app is secure by default meaning the default page is a login screen

Logged in users may define N number of boards, each board can have categories that are defined per board (categoires should have a order column for sorting first to last)

All boards start with default categories:

Backlog
Ready to Start
In Progress
Complete

Cards may be added with the following fields

Title
Description
Tag (Support pill style tags)
Due Date (Optional)
Assigned To (Set after moved from start card)
Comments - Comments by stakeholders, these are one-to-many

Because this is a project that has been done before include other features that are common to this type of program.



Use golangs built in mux for http routing.
