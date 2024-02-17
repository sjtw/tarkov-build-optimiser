package importers

import (
	"errors"
	"reflect"
	"tarkov-build-optimiser/internal/models"
)

// properties can be a variety of types due to it being a graphql union type
// using reflection to create our Slots from
// *tarkovdev.GetWeaponModsItemsItemPropertiesItemPropertiesWeaponMod
// *tarkovdev.GetWeaponModsItemsItemPropertiesItemPropertiesScope
// *tarkovdev.GetWeaponModsItemsItemPropertiesItemPropertiesBarrel
// *tarkovdev.GetWeaponModsItemsItemPropertiesItemPropertiesMagazine
func convertPropertiesToSlots(properties interface{}) ([]models.Slot, error) {
	propertiesValue := reflect.ValueOf(properties).Elem()
	slots := propertiesValue.FieldByName("Slots")
	if slots.Kind() != reflect.Slice {
		return nil, errors.New("slots is not a slice")
	}

	var newSlots []models.Slot
	for i := 0; i < slots.Len(); i++ {
		slot := slots.Index(i)

		id := slot.FieldByName("Id").String()
		name := slot.FieldByName("Name").String()

		newSlot := models.Slot{ID: id, Name: name}
		filters := slot.FieldByName("Filters")

		allowedItems := filters.FieldByName("AllowedItems")
		for j := 0; j < allowedItems.Len(); j++ {
			item := allowedItems.Index(j)
			id := item.FieldByName("Id").String()
			name := item.FieldByName("Name").String()
			newItem := models.AllowedItem{ID: id, Name: name}
			newSlot.AllowedItems = append(newSlot.AllowedItems, newItem)
		}
		newSlots = append(newSlots, newSlot)
	}

	return newSlots, nil
}
