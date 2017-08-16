package main

var CatFacts = [...]string{
	`Did you know, a 2007 Gallup poll revealed that both men and women were equally likely to own a cat.`,
	`Did you know, a cat almost never meows at another cat, mostly just humans. Cats typically will spit, purr, and hiss at other cats.`,
	`Did you know, a cat called Dusty has the known record for the most kittens. She had more than 420 kittens in her lifetime.`,
	`Did you know, a cat can jump up to five times its own height in a single bound.`,
	`Did you know, a cat can travel at a top speed of approximately 31 mph (49 km) over a short distance.`,
	`Did you know, a cat cannot climb head first down a tree because its claws are curved the wrong way A cat can’t climb head first down a tree because every claw on a cat’s paw points the same way. To get down from a tree, a cat must back down.`,
	`Did you know, a cat has 230 bones in its body. A human has 206. A cat has no collarbone, so it can fit through any opening the size of its head.`,
	`Did you know, a cat lover is called an Ailurophilia (Greek: cat+lover).`,
	`Did you know, a cat rubs against people to mark them as their territory A cat rubs against people not only to be affectionate but also to mark out its territory with scent glands around its face. The tail area and paws also carry the cat’s scent.`,
	`Did you know, a cat usually has about 12 whiskers on each side of its face.`,
	`Did you know, a cat's field of vision does not cover the area right under its nose.`,
	`Did you know, a cat’s back is extremely flexible because it has up to 53 loosely fitting vertebrae. Humans only have 34.`,
	`Did you know, a cat’s brain is biologically more similar to a human brain than it is to a dog’s. Both humans and cats have identical regions in their brains that are responsible for emotions.`,
	`Did you know, a cat’s eyesight is both better and worse than humans. It is better because cats can see in much dimmer light and they have a wider peripheral view. It’s worse because they don’t see color as well as humans do. Scientists believe grass appears red to cats.`,
	`Did you know, a cat’s hearing is better than a dog’s. And a cat can hear high-frequency sounds up to two octaves higher than a human.`,
	`Did you know, a cat’s heart beats nearly twice as fast as a human heart, at 110 to 140 beats a minute.`,
	`Did you know, a cat’s jaw can’t move sideways, so a cat can’t chew large chunks of food.`,
	`Did you know, a cat’s nose pad is ridged with a unique pattern, just like the fingerprint of a human.`,
	`Did you know, a commemorative tower was built in Scotland for a cat named Towser, who caught nearly 30,000 mice in her lifetime.`,
	`Did you know, a female cat is called a queen or a molly.`,
	`Did you know, a group of cats is called a “clowder.”`,
	`Did you know, a group of kittens is called a "kindle", and "clowder" is a term that refers to a group of adult cats.`,
	`Did you know, according to Hebrew legend, Noah prayed to God for help protecting all the food he stored on the ark from being eaten by rats. In reply, God made the lion sneeze, and out popped a cat.`,
	`Did you know, according to the Association for Pet Obesity Prevention (APOP), about 50 million of our cats are overweight.`,
	`Did you know, all cats have claws, and all except the cheetah sheath them when at rest.`,
	`Did you know, ancient Egyptians first adored cats for their finesse in killing rodents—as far back as 4,000 years ago.`,
	`Did you know, approximately 24 cat skins can make a coat.`,
	`Did you know, approximately 40,000 people are bitten by cats in the U.S. annually.`,
	`Did you know, cat ownership may improve psychological health by providing emotional support and dispelling feelings of depression, anxiety and loneliness?`,
	`Did you know, cats are extremely sensitive to vibrations. Cats are said to detect earthquake tremors 10 or 15 minutes before humans can.`,
	`Did you know, cats are the most popular pet in North American Cats are North America’s most popular pets: there are 73 million cats compared to 63 million dogs. Over 30% of households in North America own a cat.`,
	`Did you know, cats are unable to detect sweetness in anything they taste.`,
	`Did you know, cats CAN be lefties and righties, just like us. More than forty percent of them are, leaving some ambidextrous.`,
	`Did you know, cats hate the water because their fur does not insulate well when it’s wet. The Turkish Van, however, is one cat that likes swimming. Bred in central Asia, its coat has a unique texture that makes it water resistant.`,
	`Did you know, cats have 32 muscles that control the outer ear (humans have only 6). A cat can independently rotate its ears 180 degrees.`,
	`Did you know, cats have a strong aversion to anything citrus.`,
	`Did you know, cats have about 130,000 hairs per square inch (20,155 hairs per square centimeter).`,
	`Did you know, cats make about 100 different sounds. Dogs make only about 10.`,
	`Did you know, cats spend nearly 1/3 of their waking hours cleaning themselves.`,
	`Did you know, cats use their tails balance and have nearly`,
	`Did you know, cats use their whiskers to measure openings, indicate mood and general navigation.`,
	`Did you know, cats were domesticated from the African wildcat somewhere in the Middle East 10,000 years ago?`,
	`Did you know, cat’s sweat only through their paws Cats don’t have sweat glands over their bodies like humans do. Instead, they sweat only through their paws.`,
	`Did you know, during the Middle Ages, cats were associated with withcraft, and on St. John’s Day, people all over Europe would stuff them into sacks and toss the cats into bonfires. On holy days, people celebrated by tossing cats from church towers.`,
	`Did you know, during the nearly 18 hours a day that kittens sleep, an important growth hormone is released One reason that kittens sleep so much is because a growth hormone is released only during sleep.`,
	`Did you know, every year, nearly four million cats are eaten in Asia.`,
	`Did you know, female cats tend to be right pawed, while male cats are more often left pawed. Interestingly, while 90% of humans are right handed, the remaining 10% of lefties also tend to be male.`,
	`Did you know, grown cats have 30 teeth. Kittens have about 26 temporary teeth, which they lose when they are about 6 months old.`,
	`Did you know, if they have ample water, cats can tolerate temperatures up to 133 °F.`,
	`Did you know, if your cat's eyes are closed, it's not necessarily because it's tired. A sign of closed eyes means your cat is happy or pleased.`,
	`Did you know, in 1888, more than 300,000 mummified cats were found an Egyptian cemetery. They were stripped of their wrappings and carted off to be used by farmers in England and the U.S. for fertilizer.`,
	`Did you know, in contrast to dogs, cats have not undergone major changes during their domestication process.`,
	`Did you know, in Japan, cats are thought to have the power to turn into super spirits when they die. This may be because according to the Buddhist religion, the body of the cat is the temporary resting place of very spiritual people.`,
	`Did you know, in just seven years, a single pair of cats and their offspring could produce a staggering total of 420,000 kittens.`,
	`Did you know, in the 1750s, Europeans introduced cats into the Americas to control pests.`,
	`Did you know, in the original Italian version of Cinderella, the benevolent fairy godmother figure was a cat.`,
	`Did you know, isaac Newton invented the cat flap. Newton was experimenting in a pitch-black room. Spithead, one of his cats, kept opening the door and wrecking his experiment. The cat flap kept both Newton and Spithead happy.`,
	`Did you know, landing on all fours is something typical to cats thanks to the help of their eyes and special balance organs in their inner ear. These tools help them straighten themselves in the air and land upright on the ground.`,
	`Did you know, many cat owners think their cats can read their minds Approximately 1/3 of cat owners think their pets are able to read their minds.`,
	`Did you know, many Egyptians worshipped the goddess Bast, who had a woman’s body and a cat’s head.`,
	`Did you know, mohammed loved cats and reportedly his favorite cat, Muezza, was a tabby. Legend says that tabby cats have an “M” for Mohammed on top of their heads because Mohammad would often rest his hand on the cat’s head.`,
	`Did you know, most cats give birth to a litter of between one and nine kittens. The largest known litter ever produced was 19 kittens, of which 15 survived.`,
	`Did you know, most cats had short hair until about 100 years ago, when it became fashionable to own cats and experiment with breeding.`,
	`Did you know, on average, cats spend 2/3 of every day sleeping. That means a nine-year-old cat has been awake for only three years of its life.`,
	`Did you know, perhaps the most famous comic cat is the Cheshire Cat in Lewis Carroll’s Alice in Wonderland. With the ability to disappear, this mysterious character embodies the magic and sorcery historically associated with cats.`,
	`Did you know, relative to its body size, the clouded leopard has the biggest canines of all animals’ canines. Its dagger-like teeth can be as long as 1.8 inches (4.5 cm).`,
	`Did you know, researchers are unsure exactly how a cat purrs. Most veterinarians believe that a cat purrs by vibrating vocal folds deep in the throat. To do this, a muscle in the larynx opens and closes the air passage about 25 times per second.`,
	`Did you know, researchers believe the word “tabby” comes from Attabiyah, a neighborhood in Baghdad, Iraq. Tabbies got their name because their striped coats resembled the famous wavy patterns in the silk produced in this city.`,
	`Did you know, smuggling a cat out of ancient Egypt was punishable by death. Phoenician traders eventually succeeded in smuggling felines, which they sold to rich people in Athens and other important cities.`,
	`Did you know, some cats have survived falls of over 65 feet (20 meters), due largely to their “righting reflex.” The eyes and balance organs in the inner ear tell it where it is in space so the cat can land on its feet. Even cats without a tail have this ability.`,
	`Did you know, spanish-Jewish folklore recounts that Adam’s first wife, Lilith, became a black vampire cat, sucking the blood from sleeping babies. This may be the root of the superstition that a cat will smother a sleeping baby or suck out the child’s breath.`,
	`Did you know, the ability of a cat to find its way home is called “psi-traveling.” Experts think cats either use the angle of the sunlight to find their way or that cats have magnetized cells in their brains that act as compasses.`,
	`Did you know, the biggest wildcat today is the Siberian Tiger. It can be more than 12 feet (3.6 m) long (about the size of a small car) and weigh up to 700 pounds (317 kg).`,
	`Did you know, the cat who holds the record for the longest non-fatal fall is Andy. He fell from the 16th floor of an apartment building (about 200 ft/.06 km) and survived.`,
	`Did you know, the claws on the cat’s back paws aren’t as sharp as the claws on the front paws because the claws in the back don’t retract and, consequently, become worn.`,
	`Did you know, the costliest cat ever is named Little Nicky, who cost his owner $50,000. He is a clone of an older cat.`,
	`Did you know, the earliest ancestor of the modern cat lived about 30 million years ago. Scientists called it the Proailurus, which means “first cat” in Greek. The group of animals that pet cats belong to emerged around 12 million years ago.`,
	`Did you know, the Egyptian Mau is probably the oldest breed of cat. In fact, the breed is so ancient that its name is the Egyptian word for “cat.”`,
	`Did you know, the first cartoon cat was Felix the Cat in 1919. In 1940, Tom and Jerry starred in the first theatrical cartoon “Puss Gets the Boot.” In 1981 Andrew Lloyd Weber created the musical Cats, based on T.S. Eliot’s Old Possum’s Book of Practical Cats.`,
	`Did you know, the first cat in space was a French cat named Felicette (a.k.a. “Astrocat”) In 1963, France blasted the cat into outer space. Electrodes implanted in her brains sent neurological signals back to Earth. She survived the trip.`,
	`Did you know, the first cat show was organized in 1871 in London. Cat shows later became a worldwide craze.`,
	`Did you know, the group of words associated with cat (catt, cath, chat, katze) stem from the Latin catus, meaning domestic cat, as opposed to feles, or wild cat.`,
	`Did you know, the heaviest cat on record is Himmy, a Tabby from Queensland, Australia. He weighed nearly 47 pounds (21 kg). He died at the age of 10.`,
	`Did you know, the largest cat breed is the Ragdoll. Male Ragdolls weigh between 12 and 20 lbs (5.4-9.0 k). Females weigh between 10 and 15 lbs (4.5-6.8 k).`,
	`Did you know, the lightest cat on record is a blue point Himalayan called Tinker Toy, who weighed 1 pound, 6 ounces (616 g). Tinker Toy was 2.75 inches (7 cm) tall and 7.5 inches (19 cm) long.`,
	`Did you know, the little tufts of hair in a cat’s ear that help keep out dirt direct sounds into the ear, and insulate the ears are called “ear furnishings.”`,
	`Did you know, the most expensive cat was an Asian Leopard cat (ALC)-Domestic Shorthair (DSH) hybrid named Zeus. Zeus, who is 90% ALC and 10% DSH, has an asking price of £100,000 ($154,000).`,
	`Did you know, the most popular pedigreed cat is the Persian cat, followed by the Main Coon cat and the Siamese cat.`,
	`Did you know, the most traveled cat is Hamlet, who escaped from his carrier while on a flight. He hid for seven weeks behind a pane. By the time he was discovered, he had traveled nearly 373,000 miles (600,000 km).`,
	`Did you know, the normal body temperature of a cat is between 100.5 ° and 102.5 °F. A cat is sick if its temperature goes below 100 ° or above 103 °F.`,
	`Did you know, the oldest cat breed on record is the Egyptian Mau, which is also the Egyptian language's word for cat.`,
	`Did you know, the oldest cat on record was Crème Puff from Austin, Texas, who lived from 1967 to August 6, 2005, three days after her 38th birthday. A cat typically can live up to 20 years, which is equivalent to about 96 human years.`,
	`Did you know, the oldest cat to give birth was Kitty who, at the age of 30, gave birth to two kittens. During her life, she gave birth to 218 kittens.`,
	`Did you know, the richest cat is Blackie who was left £15 million by his owner, Ben Rea.`,
	`Did you know, the smallest pedigreed cat is a Singapura, which can weigh just 4 lbs (1.8 kg), or about five large cans of cat food. The largest pedigreed cats are Maine Coon cats, which can weigh 25 lbs (11.3 kg), or nearly twice as much as an average cat weighs.`,
	`Did you know, the smallest wildcat today is the Black-footed cat. The females are less than 20 inches (50 cm) long and can weigh as little as 2.5 lbs (1.2 kg).`,
	`Did you know, the technical term for a cat’s hairball is a “bezoar.”`,
	`Did you know, the tiniest cat on record is Mr. Pebbles, a 2-year-old cat that weighed 3 lbs (1.3 k) and was 6.1 inches (15.5 cm) high.`,
	`Did you know, there are more than 500 million domestic cats in the world, with approximately 40 recognized breeds.`,
	`Did you know, there are up to 60 million feral cats in the United States alone.`,
	`Did you know, unlike dogs, cats do not have a sweet tooth. Scientists believe this is due to a mutation in a key taste receptor.`,
	`Did you know, when a cat chases its prey, it keeps its head level. Dogs and humans bob their heads up and down.`,
	`Did you know, when a household cat died in ancient Egypt, its owners showed their grief by shaving their eyebrows.`,
	`Did you know, when cats piss or vomit on your shoes it's a sign of undying devotion?`,
	`Did you know, while many parts of Europe and North America consider the black cat a sign of bad luck, in Britain and Australia, black cats are considered lucky.`,
}
