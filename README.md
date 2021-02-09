# irmago-measurements

The purpose of this project is to perform measurements for the study 'IRMA and Network Anonymity with Tor' by Markus Kreukniet. This project is a modified version of the project ['irmago'](https://github.com/privacybydesign/irmago) commit ffa1dd898b6f7bf9e71493a4fe4a0b5eb95d2f55, which was committed on 5 June 2020. This project (irmago-measurements) requires the project ['irmamobile-measurements'](https://github.com/markuskreukniet/irmamobile-measurements) The setup of the irmago-measurements project should be the same as the setup of irmago.

The project irmamobile-measurements contains information in its README that also might be useful for irmago-measurements.

This README of irmago-measurements might not be complete.

## Additional Setup

This project can send emails from irmamobilemeasurementtests@gmail.com. The current configuration is in the `sendMail` function of the file `/measurementHelpers.go`. We can change this configuration to make sending from a different email possible. In this configuration is `smtpServerHost` the 'SMTP server host', `smtpServerAddress` the 'SMTP server address', and `emailAddress` the email address.

## The IRMA Server and Requestor That the Study Used

The study built this project successfully, and a result of this build is the file `/irma_server_and_requestor/irma`. We can use this file to run the same IRMA server and requestor that the study used for measurements. The study ran this IRMA server and requestor on a Ubuntu Server.
