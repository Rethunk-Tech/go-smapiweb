// Package smapiweb is a client for the web services of the Stardew Valley modding toolchain: smapi.io's mod update
// API and log parser, the SMAPI compatibility list, and the StardewModDataset.
//
// It keeps no state and makes no policy: no cache, no logging, no backoff between calls. It sends requests, bounds
// what it reads, and parses leniently, so callers decide what to remember and when to ask again.
package smapiweb
