//
// @project apfs 2018 - 2022
// @author Dmitry Ponomarev <demdxx@gmail.com> 2018 - 2022
//

package apfs

import "github.com/apfs-io/apfs/models"

// ObjectTypeByContentType value
func ObjectTypeByContentType(contentType string) ObjectType {
	return models.ObjectTypeByContentType(contentType)
}
