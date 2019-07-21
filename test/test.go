// Used in tests just keeps running forever until killed
package main

import "time"

func main() {
	<-time.After(10 * time.Second)
}
