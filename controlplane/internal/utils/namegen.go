package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Docker-style name generation using adjectives and nouns
var (
	// Adjectives that are positive, easy to remember and type
	adjectives = []string{
		"admiring", "adoring", "affectionate", "agile", "amazing",
		"awesome", "beautiful", "blissful", "bold", "brave",
		"busy", "charming", "clever", "cool", "compassionate",
		"competent", "confident", "dazzling", "determined", "eager",
		"ecstatic", "elastic", "elated", "elegant", "eloquent",
		"epic", "exciting", "fervent", "festive", "flamboyant",
		"focused", "friendly", "frosty", "funny", "gallant",
		"gifted", "goofy", "gracious", "great", "happy",
		"hardcore", "heuristic", "hopeful", "hungry", "infallible",
		"inspiring", "intelligent", "interesting", "jolly", "jovial",
		"keen", "kind", "laughing", "loving", "lucid",
		"magical", "mystifying", "modest", "musing", "naughty",
		"nervous", "nice", "nifty", "nostalgic", "objective",
		"optimistic", "peaceful", "pedantic", "pensive", "practical",
		"priceless", "quirky", "quizzical", "recursing", "relaxed",
		"reverent", "romantic", "sad", "serene", "sharp",
		"silly", "sleepy", "stoic", "strange", "stupefied",
		"suspicious", "sweet", "tender", "thirsty", "trusting",
		"unruffled", "upbeat", "vibrant", "vigilant", "vigorous",
		"wizardly", "wonderful", "xenodochial", "youthful", "zealous",
		"zen",
	}

	// Nouns that are easy to remember and type (animals, scientists, etc.)
	nouns = []string{
		"albattani", "allen", "almeida", "antonelli", "agnesi",
		"archimedes", "ardinghelli", "aryabhata", "austin", "babbage",
		"banach", "banzai", "bardeen", "bartik", "bassi",
		"beaver", "bell", "benz", "bhabha", "bhaskara",
		"blackburn", "blackwell", "bohr", "booth", "borg",
		"bose", "bouman", "boyd", "brahmagupta", "brattain",
		"brown", "buck", "burnell", "cannon", "carson",
		"cartwright", "carver", "cauchy", "cerf", "chandrasekhar",
		"chaplygin", "chatelet", "chatterjee", "chebyshev", "cohen",
		"chaum", "clarke", "colden", "cori", "cray",
		"curie", "darwin", "davinci", "dewdney", "dhawan",
		"diffie", "dijkstra", "dirac", "driscoll", "dubinsky",
		"easley", "edison", "einstein", "elbakyan", "elgamal",
		"elion", "ellis", "engelbart", "euclid", "euler",
		"faraday", "feistel", "fermat", "fermi", "feynman",
		"franklin", "gagarin", "galileo", "galois", "ganguly",
		"gates", "gauss", "germain", "goldberg", "goldstine",
		"goldwasser", "golick", "goodall", "gould", "greider",
		"grothendieck", "haibt", "hamilton", "haslett", "hawking",
		"hellman", "heisenberg", "hermann", "herschel", "hertz",
		"heyrovsky", "hodgkin", "hofstadter", "hoover", "hopper",
		"hugle", "hypatia", "ishizaka", "jackson", "jang",
		"jemison", "jennings", "jepsen", "johnson", "joliot",
		"jones", "kalam", "kapitsa", "kare", "keldysh",
		"keller", "kepler", "khayyam", "khorana", "kilby",
		"kirch", "knuth", "kowalevski", "lalande", "lamarr",
		"lamport", "leakey", "leavitt", "lederberg", "lehmann",
		"lewin", "lichterman", "liskov", "lovelace", "lumiere",
		"mahavira", "margulis", "matsumoto", "maxwell", "mayer",
		"mccarthy", "mcclintock", "mclaren", "mclean", "mcnulty",
		"mendel", "mendeleev", "meitner", "meninsky", "merkle",
		"mestorf", "mirzakhani", "moore", "morse", "murdock",
		"moser", "napier", "nash", "neumann", "newton",
		"nightingale", "nobel", "noether", "northcutt", "noyce",
		"panini", "pare", "pascal", "pasteur", "payne",
		"perlman", "pike", "poincare", "poitras", "proskuriakova",
		"ptolemy", "raman", "ramanujan", "ride", "montalcini",
		"ritchie", "rhodes", "robinson", "roentgen", "rosalind",
		"rubin", "saha", "sammet", "sanderson", "satoshi",
		"shamir", "shannon", "shaw", "shirley", "shockley",
		"shtern", "sinoussi", "snyder", "solomon", "spence",
		"stonebraker", "sutherland", "swanson", "swartz", "swirles",
		"taussig", "tereshkova", "tesla", "tharp", "thompson",
		"torvalds", "tu", "turing", "varahamihira", "vaughan",
		"visvesvaraya", "volhard", "villani", "wescoff", "wilbur",
		"wiles", "williams", "williamson", "wilson", "wing",
		"wozniak", "wright", "wu", "yalow", "yonath",
		"zhukovsky",
	}
)

// GenerateRandomName generates a random name in the format "adjective-noun"
// similar to Docker container names
func GenerateRandomName() (string, error) {
	adjIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(adjectives))))
	if err != nil {
		return "", fmt.Errorf("failed to generate random adjective index: %w", err)
	}

	nounIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(nouns))))
	if err != nil {
		return "", fmt.Errorf("failed to generate random noun index: %w", err)
	}

	return fmt.Sprintf("%s-%s", adjectives[adjIndex.Int64()], nouns[nounIndex.Int64()]), nil
}

// GenerateClusterName generates a random cluster name with the "kecs-" prefix
func GenerateClusterName() (string, error) {
	name, err := GenerateRandomName()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("kecs-%s", name), nil
}

// GenerateClusterNameWithFallback generates a random cluster name, or falls back
// to the provided name if generation fails
func GenerateClusterNameWithFallback(fallbackName string) string {
	name, err := GenerateClusterName()
	if err != nil {
		// If random generation fails, use the fallback
		return fmt.Sprintf("kecs-%s", fallbackName)
	}
	return name
}
