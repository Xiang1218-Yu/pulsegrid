package codec

var ContactCSVColumns = []string{
	"id",
	"organization_id",
	"email",
	"name",
	"company",
	"status",
	"locale",
	"timezone",
	"tags",
}

func ContactCSVColumnCount() int {
	return len(ContactCSVColumns)
}
